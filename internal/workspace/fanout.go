package workspace

import (
	"path/filepath"
	"sync"
)

// Result carries the outcome of running an operation against one module.
type Result struct {
	Module LockedModule
	Out    string
	Err    error
}

// Fanout runs fn against every module in lock concurrently. benchDir is the
// directory containing the benchmark YAML; module Path is resolved relative
// to it. Results are returned in lock-declaration order.
func Fanout(benchDir string, lock *Lock, fn func(absDir string, m LockedModule) (string, error)) []Result {
	results := make([]Result, len(lock.Modules))
	var wg sync.WaitGroup
	for i, m := range lock.Modules {
		wg.Add(1)
		go func(i int, m LockedModule) {
			defer wg.Done()
			abs := m.Path
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(benchDir, m.Path)
			}
			out, err := fn(abs, m)
			results[i] = Result{Module: m, Out: out, Err: err}
		}(i, m)
	}
	wg.Wait()
	return results
}
