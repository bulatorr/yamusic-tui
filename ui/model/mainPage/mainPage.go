package mainpage

import (
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"strings"
	"time"

	ynisonGrpc "github.com/bulatorr/go-yaynison/ynison_grpc"
	"github.com/bulatorr/go-yaynison/ynisonstate"
	"github.com/dece2183/yamusic-tui/api"
	"github.com/dece2183/yamusic-tui/config"
	"github.com/dece2183/yamusic-tui/ui/components/playlist"
	"github.com/dece2183/yamusic-tui/ui/components/search"
	"github.com/dece2183/yamusic-tui/ui/components/tracker"
	"github.com/dece2183/yamusic-tui/ui/components/tracklist"
	"github.com/dece2183/yamusic-tui/ui/style"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.design/x/clipboard"
)

type Model struct {
	program       *tea.Program
	client        *api.YaMusicClient
	width, height int

	playlist  *playlist.Model
	tracklist *tracklist.Model
	tracker   *tracker.Model
	search    *search.Model

	isSearchActive       bool
	currentPlaylistIndex int
	likedTracksMap       map[string]bool
	Ynison               *ynisonGrpc.Client
	DeviceId             string
	PlayerState          *ynisonstate.PlayerState
}

// mainpage.Model constructor.
func New() *Model {
	m := &Model{}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.program = p
	m.likedTracksMap = make(map[string]bool)

	m.playlist = playlist.New(m.program, "YaMusic")
	m.tracklist = tracklist.New(m.program, &m.likedTracksMap)
	m.tracker = tracker.New(m.program, &m.likedTracksMap)
	m.search = search.New()

	return m
}

//
// model.Model interface implementation
//

func (m *Model) Run() error {
	err := clipboard.Init()
	if err != nil {
		return err
	}

	err = m.initialLoad()
	if err != nil {
		return err
	}

	// ynison integration start
	err = m.Ynison.Connect()
	if err != nil {
		return err
	}

	// volume chan
	ps := make(chan ynisonstate.PlayingStatus)
	m.tracker.UpdatePlayingStatus = ps
	go func() {
		for i := range ps {
			if m.Ynison.IsConnected() {
				m.Ynison.UpdatePlayingStatus(&i)
			}
		}
	}()

	m.Ynison.OnMessage(func(pysr *ynisonstate.PutYnisonStateResponse) {
		m.PlayerState = pysr.GetPlayerState()
		// do anything only with current active device
		if pysr.ActiveDeviceIdOptional.GetValue() != config.Current.DeviceID {
			return
		}
		// update tracks in ynison playlist
		for i, station := range m.playlist.Items() {
			switch station.Kind {
			case playlist.YNISON:
				m.playlist.Select(i)
				list := pysr.GetPlayerState().GetPlayerQueue().GetPlayableList()
				ids := make([]string, 0)
				for _, tr := range list {
					ids = append(ids, tr.GetPlayableId())
				}

				station.CurrentTrack = int(pysr.GetPlayerState().GetPlayerQueue().GetCurrentPlayableIndex())
				station.SelectedTrack = station.CurrentTrack
				m.playlist.SetItem(i, station)
				// send request only on playlist changes
				if len(station.Tracks) != len(list) {
					data, err := m.client.Tracks(ids)
					if err != nil {
						break
					}
					station.Tracks = data
				}
				m.currentPlaylistIndex = i
				m.playlist.SetItem(i, station)

			default:

			}
		}

		// update screen
		selectedPlaylist := m.playlist.SelectedItem()
		currentPlaylist := m.playlist.Items()[m.currentPlaylistIndex]

		if selectedPlaylist.IsSame(currentPlaylist) && len(selectedPlaylist.Tracks) > 0 {
			selectedPlaylist.SelectedTrack = selectedPlaylist.CurrentTrack
			m.playlist.SetItem(m.playlist.Index(), selectedPlaylist)
		}

		tracks := make([]tracklist.Item, 0, len(selectedPlaylist.Tracks))
		for i := range selectedPlaylist.Tracks {
			track := &selectedPlaylist.Tracks[i]
			tracks = append(tracks, tracklist.NewItem(track))
		}

		m.tracklist.SetItems(tracks)
		m.tracklist.Select(selectedPlaylist.SelectedTrack)

		// changed track detection
		index := pysr.GetPlayerState().GetPlayerQueue().GetCurrentPlayableIndex()
		id := pysr.GetPlayerState().GetPlayerQueue().GetPlayableList()[index].GetPlayableId()
		if id != m.tracker.CurrentTrack().Id {
			m.playTrack(&selectedPlaylist.Tracks[selectedPlaylist.SelectedTrack])
		}

		// set progress
		status := pysr.GetPlayerState().GetStatus()
		delta := math.Abs(float64(status.GetProgressMs() - m.tracker.GetCurrentPos()))
		if delta > 5000 {
			d, err := time.ParseDuration(fmt.Sprint(status.GetProgressMs()-m.tracker.GetCurrentPos(), "ms"))
			if err == nil {
				m.tracker.Rewind(d, false)
			}
		}

		// set pause
		if status.GetPaused() {
			m.tracker.Pause(false)
		} else {
			m.tracker.Play(false)
		}

		// set volume
		for _, j := range pysr.GetDevices() {
			if j.GetInfo().GetDeviceId() == config.Current.DeviceID {
				volume := j.GetVolumeInfo().GetVolume()
				if volume != m.tracker.Volume() {
					m.tracker.SetVolume(volume)
					config.Current.Volume = volume
					config.Save()
				}
			}
		}
	})

	// ynison integration end

	_, err = m.program.Run()
	return err
}

func (m *Model) GetPlayerState() ynisonstate.PlayerState {
	return *m.PlayerState
}

func (m *Model) Send(msg tea.Msg) {
	go m.program.Send(msg)
}

//
// tea.Model interface implementation
//

func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.tracker.YnisonPlaylist = m.playlist.SelectedItem().Kind == playlist.YNISON

	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return m, tea.ClearScreen

	case tea.KeyMsg:
		controls := config.Current.Controls
		keypress := msg.String()

		switch {
		case controls.Quit.Contains(keypress):
			return m, tea.Quit
		case m.isSearchActive:
			m.search, cmd = m.search.Update(message)
			cmds = append(cmds, cmd)
		default:
			m.playlist, cmd = m.playlist.Update(message)
			cmds = append(cmds, cmd)
			m.tracklist, cmd = m.tracklist.Update(message)
			cmds = append(cmds, cmd)
			m.tracker, cmd = m.tracker.Update(message)
			cmds = append(cmds, cmd)
		}

	// playlist control update
	case playlist.Control:
		switch msg {
		case playlist.CURSOR_UP, playlist.CURSOR_DOWN:
			selectedPlaylist := m.playlist.SelectedItem()
			currentPlaylist := m.playlist.Items()[m.currentPlaylistIndex]

			if selectedPlaylist.IsSame(currentPlaylist) && len(selectedPlaylist.Tracks) > 0 {
				selectedPlaylist.SelectedTrack = selectedPlaylist.CurrentTrack
				m.playlist.SetItem(m.playlist.Index(), selectedPlaylist)
			}

			tracks := make([]tracklist.Item, 0, len(selectedPlaylist.Tracks))
			for i := range selectedPlaylist.Tracks {
				track := &selectedPlaylist.Tracks[i]
				tracks = append(tracks, tracklist.NewItem(track))
			}

			m.tracklist.SetItems(tracks)
			m.tracklist.Select(selectedPlaylist.SelectedTrack)
			if m.tracker.IsPlaying() {
				m.indicateCurrentTrackPlaying(true)
			}

			m.tracklist.Shufflable = (selectedPlaylist.Kind != playlist.NONE && selectedPlaylist.Kind != playlist.MYWAVE && len(selectedPlaylist.Tracks) > 0)
		}

	// tracklist control update
	case tracklist.Control:
		switch msg {
		case tracklist.PLAY:
			playlistItem := m.playlist.SelectedItem()
			if !playlistItem.Active {
				break
			}
			m.playSelectedPlaylist(m.tracklist.Index())
		case tracklist.CURSOR_UP, tracklist.CURSOR_DOWN:
			currentPlaylist := m.playlist.SelectedItem()
			cursorIndex := m.tracklist.Index()
			currentPlaylist.SelectedTrack = cursorIndex
			m.playlist.SetItem(m.playlist.Index(), currentPlaylist)
		case tracklist.LIKE:
			cmd = m.likeSelectedTrack()
			cmds = append(cmds, cmd)
		case tracklist.SEARCH:
			m.isSearchActive = true
		case tracklist.SHUFFLE:
			selectedPlaylist := m.playlist.SelectedItem()
			currentPlaylist := m.playlist.Items()[m.currentPlaylistIndex]

			currentTrackIndex := selectedPlaylist.CurrentTrack
			selectedTrackIndex := selectedPlaylist.SelectedTrack
			currentTrack := selectedPlaylist.Tracks[currentTrackIndex]
			selectedTrack := selectedPlaylist.Tracks[selectedTrackIndex]

			if selectedPlaylist.Kind == playlist.NONE || selectedPlaylist.Kind == playlist.MYWAVE || len(selectedPlaylist.Tracks) == 0 {
				break
			}

			tracks := make([]api.Track, len(selectedPlaylist.Tracks))
			trackList := make([]tracklist.Item, len(selectedPlaylist.Tracks))
			perm := rand.Perm(len(tracks))

			for i, v := range perm {
				tracks[v] = selectedPlaylist.Tracks[i]
				trackList[v] = tracklist.NewItem(&tracks[v])
				if currentTrack.Id == tracks[v].Id {
					currentTrackIndex = v
				}
				if selectedTrackIndex > 0 && selectedTrack.Id == tracks[v].Id {
					selectedTrackIndex = v
				}
			}

			selectedPlaylist.Tracks = tracks
			selectedPlaylist.SelectedTrack = selectedTrackIndex
			selectedPlaylist.CurrentTrack = currentTrackIndex
			m.playlist.SetItem(m.playlist.Index(), selectedPlaylist)
			m.tracklist.SetItems(trackList)
			m.tracklist.Select(selectedTrackIndex)

			if selectedPlaylist.IsSame(currentPlaylist) && m.tracker.IsPlaying() {
				m.indicateCurrentTrackPlaying(true)
			}
		case tracklist.SHARE:
			track := m.tracklist.SelectedItem().Track
			link := fmt.Sprintf("https://music.yandex.ru/album/%d/track/%s", track.Albums[0].Id, track.Id)
			clipboard.Write(clipboard.FmtText, []byte(link))
		}

	// player control update
	case tracker.Control:
		switch msg {
		case tracker.NEXT:
			m.nextTrack()
		case tracker.PREV:
			m.prevTrack()
		case tracker.LIKE:
			cmd = m.likePlayingTrack()
			cmds = append(cmds, cmd)
		}

		m.tracker, cmd = m.tracker.Update(message)
		cmds = append(cmds, cmd)

	// search control update
	case search.Control:
		switch msg {
		case search.SELECT:
			req, ok := m.search.SuggestionValue()
			if ok {
				searchRes, err := m.client.Search(req, api.SEARCH_ALL)
				if err == nil {
					cmd = m.displaySearchResults(searchRes)
					cmds = append(cmds, cmd)
				}
			}
			m.isSearchActive = false
		case search.CANCEL:
			m.isSearchActive = false
		case search.UPDATE_SUGGESTIONS:
			suggestions, err := m.client.SearchSuggest(m.search.InputValue())
			if err != nil {
				break
			}
			m.search.SetSuggestions(suggestions.Best.Text, suggestions.Suggestions)
		}

	default:
		if m.isSearchActive {
			m.search, cmd = m.search.Update(message)
			cmds = append(cmds, cmd)
		} else {
			m.playlist, cmd = m.playlist.Update(message)
			cmds = append(cmds, cmd)
			m.tracklist, cmd = m.tracklist.Update(message)
			cmds = append(cmds, cmd)
			m.tracker, cmd = m.tracker.Update(message)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if m.isSearchActive {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.search.View())
	}

	sidePanel := style.SideBoxStyle.Render(m.playlist.View())
	midPanel := lipgloss.JoinVertical(lipgloss.Left, style.TrackBoxStyle.Render(m.tracklist.View()), m.tracker.View())
	return lipgloss.JoinHorizontal(lipgloss.Bottom, sidePanel, midPanel)
}

//
// private methods
//

func (m *Model) resize(width, height int) {
	m.width, m.height = width, height
	m.playlist.SetSize(style.PlaylistsSidePanelWidth, height-4)
	m.tracklist.SetSize(m.width-m.playlist.Width()-4, height-14)
	m.tracker.SetWidth(m.width - m.playlist.Width() - 4)

	searchWidth := style.SearchModalWidth
	if searchWidth > width {
		searchWidth = width - 2
	}
	m.search.SetSize(searchWidth, height-4)
}

func (m *Model) initialLoad() error {
	var err error
	if len(config.Current.Token) == 0 {
		return fmt.Errorf("wrong token")
	}

	m.client, err = api.NewClient(config.Current.Token)
	if err != nil {
		if _, ok := err.(*url.Error); ok {
			return fmt.Errorf("unable to connect to the Yandex server")
		} else {
			return err
		}
	}

	// ynison integration start
	// replace default full state request
	request := new(ynisonstate.PutYnisonStateRequest)
	request.Parameters = &ynisonstate.PutYnisonStateRequest_UpdateFullState{
		UpdateFullState: &ynisonstate.UpdateFullState{
			PlayerState: &ynisonstate.PlayerState{
				PlayerQueue: &ynisonstate.PlayerQueue{
					CurrentPlayableIndex: -1,
					EntityType:           ynisonstate.PlayerQueue_VARIOUS,
					Options: &ynisonstate.PlayerStateOptions{
						RepeatMode: ynisonstate.PlayerStateOptions_NONE,
					},
					EntityContext: ynisonstate.PlayerQueue_BASED_ON_ENTITY_BY_DEFAULT,
				},
				Status: &ynisonstate.PlayingStatus{
					Paused:        true,
					PlaybackSpeed: 1,
				},
			},
			Device: &ynisonstate.UpdateDevice{
				Info: &ynisonstate.DeviceInfo{
					DeviceId: config.Current.DeviceID,
					Type:     ynisonstate.DeviceType_WEB,
					Title:    config.Current.DeviceName,
					AppName:  "Chrome",
				},
				VolumeInfo: &ynisonstate.DeviceVolume{
					Volume: config.Current.Volume,
				},
				Capabilities: &ynisonstate.DeviceCapabilities{
					CanBePlayer:           true,
					CanBeRemoteController: false,
					VolumeGranularity:     uint32(1 / config.Current.VolumeStep),
				},
			},
		},
	}
	m.Ynison = ynisonGrpc.NewClient(config.Current.Token, config.Current.DeviceID)
	m.Ynison.ConfigMessage = request
	// ynison integration end

	for i, station := range m.playlist.Items() {
		switch station.Kind {
		case playlist.MYWAVE:
			tracks, err := m.client.StationTracks(api.MyWaveId, nil)
			if err != nil {
				continue
			}

			station.StationId = tracks.Id
			station.StationBatch = tracks.BatchId
			for _, t := range tracks.Sequence {
				station.Tracks = append(station.Tracks, t.Track)
			}
			m.playlist.SetItem(i, station)
		case playlist.LIKES:
			likes, err := m.client.LikedTracks()
			if err != nil {
				continue
			}

			likedTracksId := make([]string, len(likes))
			for l, track := range likes {
				m.likedTracksMap[track.Id] = true
				likedTracksId[l] = track.Id
			}

			likedTracks, err := m.client.Tracks(likedTracksId)
			if err != nil {
				continue
			}

			station.Tracks = likedTracks
			m.playlist.SetItem(i, station)
		default:
		}
	}

	playlists, err := m.client.ListPlaylists()
	if err == nil {
		for _, pl := range playlists {
			playlistTracks, err := m.client.PlaylistTracks(pl.Kind, false)
			if err != nil {
				continue
			}

			m.playlist.InsertItem(-1, playlist.Item{
				Name:    pl.Title,
				Kind:    pl.Kind,
				Active:  true,
				Subitem: true,
				Tracks:  playlistTracks,
			})
		}
	}

	m.playlist.Select(0)
	m.Send(playlist.CURSOR_UP)

	return nil
}

func (m *Model) prevTrack() {
	currentPlaylist := m.playlist.Items()[m.currentPlaylistIndex]

	if currentPlaylist.CurrentTrack == 0 {
		m.Send(tracker.STOP)
		return
	}

	m.indicateCurrentTrackPlaying(false)

	currentPlaylist.CurrentTrack--
	m.playlist.SetItem(m.currentPlaylistIndex, currentPlaylist)
	m.playTrack(&currentPlaylist.Tracks[currentPlaylist.CurrentTrack])

	// ynison send current track id
	if m.tracker.YnisonPlaylist && m.PlayerState != nil && m.PlayerState.PlayerQueue != nil {
		m.PlayerState.PlayerQueue.CurrentPlayableIndex = int32(currentPlaylist.CurrentTrack)
		m.PlayerState.Status.ProgressMs = 0
		m.PlayerState.Status.DurationMs = int64(currentPlaylist.Tracks[currentPlaylist.CurrentTrack].DurationMs)
		m.Ynison.UpdatePlayerState(m.PlayerState)
	}
	// end

	selectedPlaylist := m.playlist.SelectedItem()
	if currentPlaylist.IsSame(selectedPlaylist) && m.tracklist.Index() == currentPlaylist.CurrentTrack+1 {
		m.tracklist.Select(currentPlaylist.CurrentTrack)
	}
}

func (m *Model) nextTrack() {
	currentPlaylist := m.playlist.Items()[m.currentPlaylistIndex]

	if len(currentPlaylist.Tracks) == 0 {
		return
	}

	m.indicateCurrentTrackPlaying(false)

	if currentPlaylist.Infinite {
		currTrack := currentPlaylist.Tracks[currentPlaylist.CurrentTrack]

		if m.tracker.Progress() == 1 {
			go m.client.StationFeedback(
				api.ROTOR_TRACK_FINISHED,
				currentPlaylist.StationId,
				currentPlaylist.StationBatch,
				currTrack.Id,
				currTrack.DurationMs*1000,
			)
		} else {
			go m.client.StationFeedback(
				api.ROTOR_SKIP,
				currentPlaylist.StationId,
				currentPlaylist.StationBatch,
				currTrack.Id,
				int(float64(currTrack.DurationMs*1000)*m.tracker.Progress()),
			)
		}

		if currentPlaylist.CurrentTrack+2 >= len(currentPlaylist.Tracks) {
			tracks, err := m.client.StationTracks(api.MyWaveId, &currTrack)
			if err != nil {
				return
			}

			for _, tr := range tracks.Sequence {
				// automatic append new tracks to the track list if this playlist is selected
				currentPlaylist.Tracks = append(currentPlaylist.Tracks, tr.Track)
				if m.playlist.SelectedItem().IsSame(currentPlaylist) {
					newTrack := &currentPlaylist.Tracks[len(currentPlaylist.Tracks)-1]
					m.tracklist.InsertItem(-1, tracklist.NewItem(newTrack))
				}
			}
		}
	} else if currentPlaylist.CurrentTrack+1 >= len(currentPlaylist.Tracks) {
		currentPlaylist.CurrentTrack = 0
		m.playlist.SetItem(m.currentPlaylistIndex, currentPlaylist)
		m.Send(tracker.STOP)
		return
	}

	currentPlaylist.CurrentTrack++
	m.playlist.SetItem(m.currentPlaylistIndex, currentPlaylist)
	m.playTrack(&currentPlaylist.Tracks[currentPlaylist.CurrentTrack])

	// ynison send current track id
	if m.tracker.YnisonPlaylist && m.PlayerState != nil && m.PlayerState.PlayerQueue != nil {
		m.PlayerState.PlayerQueue.CurrentPlayableIndex = int32(currentPlaylist.CurrentTrack)
		m.PlayerState.Status.ProgressMs = 0
		m.PlayerState.Status.DurationMs = int64(currentPlaylist.Tracks[currentPlaylist.CurrentTrack].DurationMs)
		m.Ynison.UpdatePlayerState(m.PlayerState)
	}
	// end

	selectedPlaylist := m.playlist.SelectedItem()
	if currentPlaylist.IsSame(selectedPlaylist) && m.tracklist.Index() == currentPlaylist.CurrentTrack-1 {
		m.tracklist.Select(currentPlaylist.CurrentTrack)
	}
}

func (m *Model) playTrack(track *api.Track) {
	m.tracker.Stop()

	dowInfo, err := m.client.TrackDownloadInfo(track.Id)
	if err != nil {
		return
	}

	var bestBitrate int
	var bestTrackInfo api.TrackDownloadInfo
	for _, t := range dowInfo {
		if t.BbitrateInKbps > bestBitrate {
			bestBitrate = t.BbitrateInKbps
			bestTrackInfo = t
		}
	}

	trackReader, _, err := m.client.DownloadTrack(bestTrackInfo)
	if err != nil {
		return
	}

	m.indicateCurrentTrackPlaying(true)
	m.tracker.StartTrack(track, trackReader)

	currentPlaylist := m.playlist.Items()[m.currentPlaylistIndex]
	if currentPlaylist.Infinite {
		go m.client.StationFeedback(
			api.ROTOR_TRACK_STARTED,
			currentPlaylist.StationId,
			currentPlaylist.StationBatch,
			track.Id,
			0,
		)
	}

	go m.client.PlayTrack(track, false)
}

func (m *Model) playSelectedPlaylist(trackIndex int) {
	currentPlaylist := m.playlist.Items()[m.currentPlaylistIndex]
	selectedPlaylist := m.playlist.SelectedItem()
	trackToPlay := &selectedPlaylist.Tracks[selectedPlaylist.SelectedTrack]

	if len(currentPlaylist.Tracks) == 0 {
		m.Send(tracker.STOP)
		return
	}

	if currentPlaylist.IsSame(selectedPlaylist) && m.tracker.CurrentTrack() == trackToPlay {
		if m.tracker.IsPlaying() {
			m.tracker.Pause(true)
			return
		} else {
			m.tracker.Play(true)
			return
		}
	}

	m.indicateCurrentTrackPlaying(false)
	selectedPlaylist.CurrentTrack = trackIndex

	if selectedPlaylist.Infinite {
		if m.tracker.IsPlaying() {
			currentTrack := m.tracker.CurrentTrack()
			go m.client.StationFeedback(
				api.ROTOR_SKIP,
				selectedPlaylist.StationId,
				selectedPlaylist.StationBatch,
				currentTrack.Id,
				int(float64(currentTrack.DurationMs*1000)*m.tracker.Progress()),
			)
			go m.client.StationFeedback(
				api.ROTOR_TRACK_STARTED,
				selectedPlaylist.StationId,
				selectedPlaylist.StationBatch,
				trackToPlay.Id,
				0,
			)
		} else {
			go m.client.StationFeedback(
				api.ROTOR_RADIO_STARTED,
				selectedPlaylist.StationId,
				"",
				"",
				0,
			)
		}
	}

	m.currentPlaylistIndex = m.playlist.Index()
	m.playlist.SetItem(m.currentPlaylistIndex, selectedPlaylist)

	// ynison send current track id
	if m.tracker.YnisonPlaylist && m.PlayerState != nil && m.PlayerState.PlayerQueue != nil {
		m.PlayerState.PlayerQueue.CurrentPlayableIndex = int32(trackIndex)
		m.PlayerState.Status.ProgressMs = 0
		m.PlayerState.Status.DurationMs = int64(currentPlaylist.Tracks[trackIndex].DurationMs)
		m.Ynison.UpdatePlayerState(m.PlayerState)
	}
	// end

	m.playTrack(trackToPlay)
}

func (m *Model) likePlayingTrack() tea.Cmd {
	track := m.tracker.CurrentTrack()
	return m.likeTrack(track)
}

func (m *Model) likeSelectedTrack() tea.Cmd {
	currentPlaylist := m.playlist.Items()[m.currentPlaylistIndex]
	if len(currentPlaylist.Tracks) == 0 {
		return nil
	}

	track := m.tracklist.SelectedItem().Track
	return m.likeTrack(track)
}

func (m *Model) likeTrack(track *api.Track) tea.Cmd {
	if m.likedTracksMap[track.Id] {
		if m.client.UnlikeTrack(track.Id) != nil {
			return nil
		}

		delete(m.likedTracksMap, track.Id)

		likedPlaylist := m.playlist.Items()[1]
		for i, ltrack := range likedPlaylist.Tracks {
			if ltrack.Id == track.Id {
				if i+1 < len(likedPlaylist.Tracks) {
					likedPlaylist.Tracks = append(likedPlaylist.Tracks[:i], likedPlaylist.Tracks[i+1:]...)
				} else {
					likedPlaylist.Tracks = likedPlaylist.Tracks[:i]
				}

				if m.playlist.SelectedItem().Kind == playlist.LIKES {
					m.tracklist.RemoveItem(i)
				}
				break
			}
		}

		return m.playlist.SetItem(1, likedPlaylist)
	} else {
		if m.client.LikeTrack(track.Id) != nil {
			return nil
		}

		m.likedTracksMap[track.Id] = true
		likedPlaylist := m.playlist.Items()[1]
		likedPlaylist.Tracks = append([]api.Track{*track}, likedPlaylist.Tracks...)
		return m.playlist.SetItem(1, likedPlaylist)
	}
}

func (m *Model) indicateCurrentTrackPlaying(playing bool) {
	currentPlaylist := m.playlist.Items()[m.currentPlaylistIndex]
	if currentPlaylist.IsSame(m.playlist.SelectedItem()) && currentPlaylist.CurrentTrack < len(m.tracklist.Items()) {
		track := m.tracklist.Items()[currentPlaylist.CurrentTrack]
		track.IsPlaying = playing
		m.tracklist.SetItem(currentPlaylist.CurrentTrack, track)
	}
}

func (m *Model) displaySearchResults(res api.SearchResult) tea.Cmd {
	playlists := m.playlist.Items()
	searchResIndex := len(playlists) + 2
	for i, pl := range playlists {
		if !pl.Active && !pl.Subitem && pl.Name == "search results:" {
			playlists = playlists[:i-1]
			searchResIndex = i + 1
			break
		}
	}

	playlists = append(playlists,
		playlist.Item{Name: "", Kind: playlist.NONE, Active: false, Subitem: false},
		playlist.Item{Name: "search results:", Kind: playlist.NONE, Active: false, Subitem: false},
	)

	if len(res.Tracks.Results) > 0 {
		playlists = append(playlists, playlist.Item{
			Name:    "tracks",
			Active:  true,
			Subitem: true,
			Tracks:  res.Tracks.Results,
		})
	}

	if len(res.Artists.Results) > 0 {
		for _, artist := range res.Artists.Results {
			if !strings.Contains(strings.ToLower(artist.Name), strings.ToLower(res.Text)) {
				continue
			}

			artistTracks, err := m.client.ArtistPopularTracks(artist.Id)
			if err != nil {
				continue
			}

			tracks, err := m.client.Tracks(artistTracks.Tracks)
			if err != nil {
				continue
			}

			playlists = append(playlists, playlist.Item{
				Name:    artist.Name,
				Active:  true,
				Subitem: true,
				Tracks:  tracks,
			})
		}
	}

	cmd := m.playlist.SetItems(playlists)
	m.playlist.Select(searchResIndex)
	m.Send(playlist.CURSOR_DOWN)

	return cmd
}
