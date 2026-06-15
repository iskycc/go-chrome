//go:build !windows

package sysinfo

func (s *Sampler) chromeInfoLocked(pid int32) ProcessSnapshot {
	p, err := s.processForChromeLocked(pid)
	if err != nil {
		s.resetChromeLocked()
		return ProcessSnapshot{Exists: false}
	}
	return infoFromProcess(pid, p)
}
