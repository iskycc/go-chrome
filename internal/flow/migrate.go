package flow

// CurrentSchemaVersion is the latest flow schema version.
const CurrentSchemaVersion = 1

// Migrate upgrades an old flow to the current schema.
func Migrate(f *Flow) error {
	for f.SchemaVersion < CurrentSchemaVersion {
		switch f.SchemaVersion {
		case 0:
			migrate0To1(f)
		default:
			// unknown version, treat as current
			f.SchemaVersion = CurrentSchemaVersion
		}
	}
	return nil
}

func migrate0To1(f *Flow) {
	f.SchemaVersion = 1
	for i := range f.Steps {
		if f.Steps[i].ID == "" {
			// In a real scenario we'd assign UUIDs, but store.Import does that.
		}
		if f.Steps[i].OnError == "" {
			f.Steps[i].OnError = ErrStop
		}
		if f.Steps[i].TimeoutMs == 0 {
			f.Steps[i].TimeoutMs = 10000
		}
	}
}
