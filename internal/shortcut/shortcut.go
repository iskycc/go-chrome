package shortcut

// Options describes a Windows shortcut to create.
type Options struct {
	TargetPath   string
	Arguments    string
	WorkingDir   string
	IconPath     string
	Description  string
	ShortcutPath string
}

