package models

type Mod struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Filename    string `json:"filename"`
	Size        string `json:"size"`
	SizeBytes   int64  `json:"sizeBytes"`
	IsDLC       bool   `json:"isDLC"`
	IsActive    bool   `json:"isActive"`
	URL         string `json:"url"`
	NeedsUpdate bool   `json:"needsUpdate"`
	LocalVersion string `json:"localVersion,omitempty"`
}

type SyncStatus struct {
	TotalMods      int    `json:"totalMods"`
	ModsToSync     int    `json:"modsToSync"`
	TotalSize      string `json:"totalSize"`
	TotalSizeBytes int64  `json:"totalSizeBytes"`
	GameVersion    string `json:"gameVersion"`
}

type SyncResult struct {
	Status SyncStatus `json:"status"`
	Mods   []Mod      `json:"mods"`
}

type ValidationResult struct {
	Valid       bool   `json:"valid"`
	GameVersion string `json:"gameVersion"`
	ModCount    int    `json:"modCount"`
	Error       string `json:"error,omitempty"`
}

type ProgressEvent struct {
	Filename    string  `json:"filename"`
	Progress    float64 `json:"progress"`
	Downloaded  int     `json:"downloaded"`
	Total       int     `json:"total"`
	CurrentSize int64   `json:"currentSize"`
	TotalSize   int64   `json:"totalSize"`
	Speed       string  `json:"speed"`
}
