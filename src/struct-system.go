package src

import "xteve-reborn/src/internal/imgcache"

// SystemStruct : Beinhaltet alle Systeminformationen
type SystemStruct struct {
	Addresses struct {
		DVR string
		M3U string
		XML string
	}

	APIVersion             string
	AppName                string
	ARCH                   string
	Branch                 string
	Build                  string
	Compatibility          string
	ConfigurationWizard    bool
	DBVersion              string
	Dev                    bool
	DeviceID               string
	Domain                 string
	PlexChannelLimit       int
	UnfilteredChannelLimit int

	FFmpeg struct {
		DefaultOptions string
		Path           string
	}

	VLC struct {
		DefaultOptions string
		Path           string
	}

	File struct {
		Authentication string
		M3U            string
		PMS            string
		Settings       string
		URLS           string
		XEPG           string
		XML            string
	}

	Compressed struct {
		GZxml string
	}

	Flag struct {
		Branch  string
		Debug   int
		Info    bool
		Port    string
		Restore string
		SSDP    bool
	}

	Folder struct {
		Backup       string
		Cache        string
		Config       string
		Data         string
		ImagesCache  string
		ImagesUpload string
		Temp         string
	}

	Hostname          string
	IPAddress         string
	IPAddressesList   []string
	IPAddressesV4     []string
	IPAddressesV6     []string
	Name              string
	OS                string
	TimeForAutoUpdate string

	Notification map[string]Notification

	ServerProtocol struct {
		API string
		DVR string
		M3U string
		WEB string
		XML string
	}

	GitHub struct {
		Branch string
		Repo   string
		Update bool
		User   string
	}

	// ReleaseTag is the exact git tag this binary was built from (e.g.
	// "v3.0.0-pre.2"), baked in at build time via -ldflags for official
	// release builds. Empty for a plain `go build .` from source, which
	// disables update checking entirely - there's no reliable version to
	// compare against otherwise.
	ReleaseTag string

	// UpdateAvailable/UpdateVersion cache the result of the last update
	// check (BinaryUpdate, or the manual "Check for Updates" WS command),
	// so the dashboard can display it without triggering a GitHub API call
	// on every single WS response - those fire continuously (e.g. the
	// background log poll every 10s) and would exhaust GitHub's
	// unauthenticated rate limit almost immediately. Guarded by
	// updateStateLock.
	UpdateAvailable bool
	UpdateVersion   string

	URLBase string
	UDPxy   string
	Version string
	WEB     struct {
		Menu []string
	}
}

// DataStruct : Alle Daten werden hier abgelegt. (Lineup, XMLTV)
type DataStruct struct {
	Cache struct {
		Images      *imgcache.Cache
		ImagesCache []string
		ImagesFiles []string
		ImagesURLS  []string
		PMS         map[string]string

		StreamingURLS map[string]StreamInfo
		XMLTV         map[string]XMLTV

		Streams struct {
			Active []string
		}
	}

	Filter []Filter

	Playlist struct {
		M3U struct {
			Groups struct {
				Text  []string
				Value []string
			}
		}
	}

	StreamPreviewUI struct {
		Active   []string
		Inactive []string
	}

	Streams struct {
		Active   []interface{}
		All      []interface{}
		Inactive []interface{}
	}

	XMLTV struct {
		Files   []string
		Mapping map[string]interface{}
	}

	XEPG struct {
		Channels  map[string]interface{}
		XEPGCount int64
	}
}

// Filter : Wird für die Filterregeln verwendet
type Filter struct {
	CaseSensitive bool
	Rule          string
	Type          string
}

// XEPGChannelStruct : XEPG Struktur
type XEPGChannelStruct struct {
	FileM3UID          string `json:"_file.m3u.id,required"`
	FileM3UName        string `json:"_file.m3u.name,required"`
	FileM3UPath        string `json:"_file.m3u.path,required"`
	GroupTitle         string `json:"group-title,required"`
	Name               string `json:"name,required"`
	TvgID              string `json:"tvg-id,required"`
	TvgLogo            string `json:"tvg-logo,required"`
	TvgName            string `json:"tvg-name,required"`
	URL                string `json:"url,required"`
	UUIDKey            string `json:"_uuid.key,required"`
	UUIDValue          string `json:"_uuid.value,omitempty"`
	Values             string `json:"_values,required"`
	XActive            bool   `json:"x-active,required"`
	XCategory          string `json:"x-category,required"`
	XChannelID         string `json:"x-channelID,required"`
	XEPG               string `json:"x-epg,required"`
	XGroupTitle        string `json:"x-group-title,required"`
	XMapping           string `json:"x-mapping,required"`
	XmltvFile          string `json:"x-xmltv-file,required"`
	XName              string `json:"x-name,required"`
	XUpdateChannelIcon bool   `json:"x-update-channel-icon,required"`
	XUpdateChannelName bool   `json:"x-update-channel-name,required"`
	XDescription       string `json:"x-description,required"`
	XBackupChannel1    string `json:"x-backup-channel-1,omitempty"`
	XBackupChannel2    string `json:"x-backup-channel-2,omitempty"`
	XBackupChannel3    string `json:"x-backup-channel-3,omitempty"`
	// XDirectives carries the provider's per-stream directives (#KODIPROP,
	// #EXTVLCOPT, #EXTHTTP, ...) verbatim, newline separated, so they can be
	// emitted again in the M3U this instance serves.
	XDirectives string `json:"x-directives,omitempty"`
}

// M3UChannelStructXEPG : M3U Struktur für XEPG
type M3UChannelStructXEPG struct {
	FileM3UID   string `json:"_file.m3u.id,required"`
	FileM3UName string `json:"_file.m3u.name,required"`
	FileM3UPath string `json:"_file.m3u.path,required"`
	GroupTitle  string `json:"group-title,required"`
	Name        string `json:"name,required"`
	TvgID       string `json:"tvg-id,required"`
	TvgLogo     string `json:"tvg-logo,required"`
	TvgName     string `json:"tvg-name,required"`
	URL         string `json:"url,required"`
	UUIDKey     string `json:"_uuid.key,required"`
	UUIDValue   string `json:"_uuid.value,required"`
	Values      string `json:"_values,required"`
	// XDirectives is the provider's per-stream directives, newline separated
	// (see XEPGChannelStruct.XDirectives).
	XDirectives string `json:"_directives,omitempty"`
}

// FilterStruct : Filter Struktur
type FilterStruct struct {
	Active        bool   `json:"active,required"`
	CaseSensitive bool   `json:"caseSensitive,required"`
	Description   string `json:"description,required"`
	Exclude       string `json:"exclude,required"`
	Filter        string `json:"filter,required"`
	Include       string `json:"include,required"`
	Name          string `json:"name,required"`
	Rule          string `json:"rule,omitempty"`
	Type          string `json:"type,required"`
}

// StreamingURLS : Informationen zu allen streaming URL's
type StreamingURLS struct {
	Streams map[string]StreamInfo `json:"channels,required"`
}

// StreamInfo : Informationen zum Kanal für die streaming URL
type StreamInfo struct {
	ChannelNumber string `json:"channelNumber,required"`
	Name          string `json:"name,required"`
	PlaylistID    string `json:"playlistID,required"`
	URL           string `json:"url,required"`
	URLid         string `json:"urlID,required"`
	BackupURL1    string `json:"backupUrl1,omitempty"`
	BackupURL2    string `json:"backupUrl2,omitempty"`
	BackupURL3    string `json:"backupUrl3,omitempty"`
}

// Notification : Notifikationen im Webinterface
type Notification struct {
	Headline string `json:"headline,required"`
	Message  string `json:"message,required"`
	New      bool   `json:"new,required"`
	Time     string `json:"time,required"`
	Type     string `json:"type,required"`
}

// SettingsStruct : Inhalt der settings.json
type SettingsStruct struct {
	API               bool     `json:"api"`
	AuthenticationAPI bool     `json:"authentication.api"`
	AuthenticationM3U bool     `json:"authentication.m3u"`
	AuthenticationPMS bool     `json:"authentication.pms"`
	AuthenticationWEB bool     `json:"authentication.web"`
	AuthenticationXML bool     `json:"authentication.xml"`
	BackupKeep        int      `json:"backup.keep"`
	BackupPath        string   `json:"backup.path"`
	Branch            string   `json:"git.branch,omitempty"`
	Buffer            string   `json:"buffer"`
	BufferSize        int      `json:"buffer.size.kb"`
	BufferTimeout     float64  `json:"buffer.timeout"`
	CacheImages       bool     `json:"cache.images"`
	EpgSource         string   `json:"epgSource"`
	FFmpegOptions     string   `json:"ffmpeg.options"`
	FFmpegPath        string   `json:"ffmpeg.path"`
	VLCOptions        string   `json:"vlc.options"`
	VLCPath           string   `json:"vlc.path"`
	FileM3U           []string `json:"file,omitempty"`  // Beim Wizard wird die M3U in ein Slice gespeichert
	FileXMLTV         []string `json:"xmltv,omitempty"` // Altes Speichersystem der Provider XML Datei Slice (Wird für die Umwandlung auf das neue benötigt)

	Files struct {
		HDHR  map[string]interface{} `json:"hdhr"`
		M3U   map[string]interface{} `json:"m3u"`
		XMLTV map[string]interface{} `json:"xmltv"`
	} `json:"files"`

	FilesUpdate               bool                  `json:"files.update"`
	Filter                    map[int64]interface{} `json:"filter"`
	Key                       string                `json:"key,omitempty"`
	Language                  string                `json:"language"`
	LogEntriesRAM             int                   `json:"log.entries.ram"`
	M3U8AdaptiveBandwidthMBPS int                   `json:"m3u8.adaptive.bandwidth.mbps"`
	MappingFirstChannel       float64               `json:"mapping.first.channel"`
	PlexChannelLimit          int                   `json:"plex.channel.limit"`
	Port                      string                `json:"port"`
	SourceCheckInterval       int                   `json:"source.check.interval"`
	SSDP                      bool                  `json:"ssdp"`
	TempPath                  string                `json:"temp.path"`
	Tuner                     int                   `json:"tuner"`
	UnfilteredChannelLimit    int                   `json:"unfiltered.channel.limit"`
	Update                    []string              `json:"update"`
	UserAgent                 string                `json:"user.agent"`
	UUID                      string                `json:"uuid"`
	UDPxy                     string                `json:"udpxy"`
	Version                   string                `json:"version"`
	XepgReplaceMissingImages  bool                  `json:"xepg.replace.missing.images"`
	XteveAutoUpdate           bool                  `json:"xteveAutoUpdate"`
}

// LanguageUI : Sprache für das WebUI
type LanguageUI struct {
	Login struct {
		Failed string
	}
}
