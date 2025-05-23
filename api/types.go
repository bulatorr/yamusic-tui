package api

type fullDownloadInfo struct {
	Host string `json:"host"`
	Path string `json:"path"`
	Ts   string `json:"ts"`
	S    string `json:"s"`
}

type YaMusicClient struct {
	token  string
	userid uint64
}

type ResultError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

type InvocInfo struct {
	ExecDurationMillis int    `json:"exec-duration-millis"`
	Hostname           string `json:"hostname"`
	ReqId              string `json:"req-id"`
}

type UserStatus struct {
	Account struct {
		Uid              uint64 `json:"uid"`
		DisplayName      string `json:"displayName"`
		FirstName        string `json:"firstName"`
		SecondName       string `json:"secondName"`
		FullName         string `json:"fullName"`
		Login            string `json:"login"`
		ServiceAvailable bool   `json:"serviceAvailable"`
	} `json:"account"`

	Permissions struct {
		Until  string   `json:"until"`
		Values []string `json:"values"`
	} `json:"permissions"`

	Plus struct {
		HasPlus             bool `json:"hasPlus"`
		IsTutorialCompleted bool `json:"isTutorialCompleted"`
	} `json:"plus"`
}

type Cover struct {
	Type     string   `json:"type"`
	Uri      string   `json:"uri"`
	Dir      string   `json:"dir"`
	ItemsUri []string `json:"itemsUri"`
}

type Owner struct {
	Login    string `json:"login"`
	Name     string `json:"name"`
	Sex      string `json:"sex"`
	Uid      uint64 `json:"uid"`
	Verified bool   `json:"verified"`
}

type Tag struct {
	Id    string `json:"id"`
	Value string `json:"value"`
}

type Label struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Artist struct {
	Id       uint64   `json:"id"`
	Name     string   `json:"name"`
	Various  bool     `json:"various"`
	Composer bool     `json:"composer"`
	Cover    Cover    `json:"cover"`
	Genres   []string `json:"genres"`
}

type ArtistTracks struct {
	Artist Artist   `json:"artist"`
	Tracks []string `json:"tracks"`
}

type Album struct {
	Id          uint64    `json:"id"`
	Title       string    `json:"title"`
	Available   bool      `json:"available"`
	Type        string    `json:"type"`
	MetaType    string    `json:"metaType"`
	Year        int       `json:"year"`
	ReleaseDate string    `json:"releaseDate"`
	CoverUri    string    `json:"coverUri"`
	OgImage     string    `json:"ogImage"`
	Genre       string    `json:"genre"`
	Recent      bool      `json:"recent"`
	TrackCount  int       `json:"trackCount"`
	Volumes     [][]Track `json:"volumes"`
	Artists     []Artist  `json:"artists"`
	Labels      []Label   `json:"labels"`
}

type Track struct {
	Id              string `json:"id"`
	RealId          string `json:"realId"`
	Title           string `json:"title"`
	Version         string `json:"version"`
	Available       bool   `json:"available"`
	Type            string `json:"type"`
	CoverUri        string `json:"coverUri"`
	OgImage         string `json:"ogImage"`
	LyricsAvailable bool   `json:"lyricsAvailable"`
	LyricsInfo      struct {
		HasAvailableSyncLyrics bool `json:"hasAvailableSyncLyrics"`
		HasAvailableTextLyrics bool `json:"hasAvailableTextLyrics"`
	} `json:"lyricsInfo"`
	Normalization struct {
		Gain float32 `json:"gain"`
		Peak float32 `json:"Peak"`
	} `json:"normalization"`

	Fade struct {
		InStart  float32 `json:"inStart"`
		InStop   float32 `json:"inStop"`
		OutStart float32 `json:"outStart"`
		OutStop  float32 `json:"outStop"`
	} `json:"fade"`

	Major struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"major"`

	Artists []Artist `json:"artists"`

	Albums []Album `json:"albums"`

	FileSize         int    `json:"fileSize"`
	StorageDir       string `json:"storageDir"`
	DurationMs       int    `json:"durationMs"`
	RememberPosition bool   `json:"rememberPosition"`
}

type Playlist struct {
	Uid  uint64 `json:"uid"`
	Kind uint64 `json:"kind"`

	Title                string `json:"title"`
	Description          string `json:"description"`
	DescriptionFormatted string `json:"descriptionFormatted"`
	Available            bool   `json:"available"`
	Collective           bool   `json:"collective"`
	Created              string `json:"created"`
	Modified             string `json:"modified"`
	Visibility           string `json:"visibility"`
	LikesCount           int    `json:"likesCount"`
	Revision             int    `json:"revision"`

	Tags    []Tag  `json:"tags"`
	Owner   Owner  `json:"owner"`
	Cover   Cover  `json:"cover"`
	OgImage string `json:"ogImage"`

	BackgroundColor string `json:"backgroundColor"`
	TextColor       string `json:"textColor"`

	TrackCount int `json:"trackCount"`
	Tracks     []struct {
		Id        uint64 `json:"id"`
		PlayCount int    `json:"playCount"`
		Recent    bool   `json:"recent"`
		Timestamp string `json:"timestamp"`
		Track     Track  `json:"track"`
	} `json:"tracks"`
}

type StationId struct {
	Type string `json:"type"`
	Tag  string `json:"tag"`
}

type Station struct {
	Id   StationId `json:"id"`
	Name string    `json:"name"`

	Icon struct {
		BackgroundColor string `json:"backgroundIcon"`
		ImageUrl        string `json:"imageUrl"`
	} `json:"icon"`
	FullImageUrl string `json:"fullImageUrl"`

	Restrictions struct {
		Language struct {
			Type           string `json:"type"`
			Name           string `json:"name"`
			PossibleValues struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"possibleValues"`
		} `json:"language"`

		Mood struct {
			Type string `json:"type"`
			Name string `json:"name"`
			Min  struct {
				Name  string  `json:"name"`
				Value float32 `json:"value"`
			} `json:"min"`
			Max struct {
				Name  string  `json:"name"`
				Value float32 `json:"value"`
			} `json:"max"`
		} `json:"mood"`

		Energy struct {
			Type string `json:"type"`
			Name string `json:"name"`
			Min  struct {
				Name  string  `json:"name"`
				Value float32 `json:"value"`
			} `json:"min"`
			Max struct {
				Name  string  `json:"name"`
				Value float32 `json:"value"`
			} `json:"max"`
		} `json:"energy"`

		Diversity struct {
			Type           string `json:"type"`
			Name           string `json:"name"`
			PossibleValues struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"possibleValues"`
		} `json:"diversity"`
	} `json:"restrictions"`
}

type StationDesc struct {
	Station  Station `json:"station"`
	Settings struct {
		Language  string  `json:"language"`
		Diversity string  `json:"diversity"`
		Mood      float32 `json:"mood"`
		Energy    float32 `json:"energy"`
	} `json:"settings"`
	RupTitle       string `json:"rupTitle"`
	RupDescription string `json:"rupDescription"`
}

type StationTracks struct {
	Id       StationId `json:"id"`
	Sequence []struct {
		Type  string `json:"type"`
		Track Track  `json:"track"`
		Liked bool   `json:"liked"`
	} `json:"sequence"`
	BatchId        string `json:"batchId"`
	RadioSessionId string `json:"radioSessionId"`
}

type LikeTrackInfo struct {
	Id        string `json:"id"`
	AlbumId   string `json:"albumId"`
	Timestamp string `json:"timestamp"`
}

type LikesDesc struct {
	Library struct {
		Uid       uint64          `json:"uid"`
		Revisions int             `json:"revisions"`
		Tracks    []LikeTrackInfo `json:"tracks"`
	} `json:"library"`
}

type TrackDownloadInfo struct {
	Codec           string `json:"codec"`
	Gain            bool   `json:"gain"`
	Preview         bool   `json:"preview"`
	DownloadInfoUrl string `json:"downloadInfoUrl"`
	Direct          bool   `json:"direct"`
	BbitrateInKbps  int    `json:"bitrateInKbps"`
}

type SearchType string

const (
	SEARCH_ARTIST = "artist"
	SEARCH_ALBUM  = "album"
	SEARCH_TRACK  = "track"
	SEARCH_ALL    = "all"
)

type SearchResult struct {
	SearchResultId string `json:"searchResultId"`
	Text           string `json:"text"`
	Best           struct {
		Type   string `json:"type"`
		Text   string `json:"text"`
		Result Track  `json:"result"`
	} `json:"best"`

	Albums struct {
		Type    string  `json:"type"`
		Total   int     `json:"total"`
		PerPage int     `json:"perPage"`
		Order   int     `json:"order"`
		Results []Album `json:"results"`
	} `json:"albums"`

	Artists struct {
		Type    string   `json:"type"`
		Total   int      `json:"total"`
		PerPage int      `json:"perPage"`
		Order   int      `json:"order"`
		Results []Artist `json:"results"`
	} `json:"artists"`

	Playlists struct {
		Type    string     `json:"type"`
		Total   int        `json:"total"`
		PerPage int        `json:"perPage"`
		Order   int        `json:"order"`
		Results []Playlist `json:"results"`
	} `json:"playlists"`

	Tracks struct {
		Type    string  `json:"type"`
		Total   int     `json:"total"`
		PerPage int     `json:"perPage"`
		Order   int     `json:"order"`
		Results []Track `json:"results"`
	} `json:"tracks"`

	Type              string `json:"type"`
	Page              int    `json:"page"`
	PerPage           int    `json:"perPage"`
	MisspellCorrected bool   `json:"misspellCorrected"`
	MisspellOriginal  string `json:"misspellOriginal"`
	Nocorrect         bool   `json:"nocorrect"`
}

type SearchSuggest struct {
	Best struct {
		Type   string `json:"type"`
		Text   string `json:"text"`
		Result Track  `json:"result"`
	} `json:"best"`
	Suggestions []string `json:"suggestions"`
}

type TrackLyrics struct {
	DownloadUrl     string   `json:"downloadUrl"`
	LyricId         string   `json:"lyricId"`
	ExternalLyricId string   `json:"externalLyricId"`
	Writers         []string `json:"writers"`
	Major           struct {
		Id         int    `json:"id"`
		Name       string `json:"name"`
		PrettyName string `json:"prettyName"`
	} `json:"major"`
}

type LyricPair struct {
	Timestamp int
	Line      string
}
