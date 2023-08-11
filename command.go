package main

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"log/slog"
)

func (w *watcher) buildCmd(filename string) []string {
	// copy original command and replace placeholder with changed filename
	cmd := make([]string, len(w.command))
	copy(cmd, w.command)
	if w.placeholder != "" && w.mode != modeService {
		for i, val := range cmd {
			if strings.Contains(val, w.placeholder) {
				cmd[i] = strings.ReplaceAll(val, w.placeholder, filename)
			}
		}
	}

	if w.shell != "" {
		cmd = append([]string{w.shell, "-c"}, strings.Join(cmd, " "))
	}
	return cmd
}

func (w *watcher) runCmd(filename string) (*exec.Cmd, error) {
	cmd := w.buildCmd(filename)
	ccmd := exec.Command(cmd[0], cmd[1:]...)
	ccmd.Stderr = os.Stderr
	ccmd.Stdout = os.Stdout
	err := ccmd.Start()
	if err != nil {
		return nil, err
	}
	return ccmd, nil
}

// cmdLoop is the command runner with burst protection
func (w *watcher) cmdLoop() {
	var prevCmd *exec.Cmd
outer:
	fnames := map[string]struct{}{
		<-w.runcmd: {},
	}

inner:
	select {
	case newcmd := <-w.runcmd:
		fnames[newcmd] = struct{}{}
		goto inner
	case <-time.After(time.Millisecond):
		for fname := range fnames {
			if prevCmd != nil {
				if err := prevCmd.Process.Signal(syscall.SIGTERM); err != nil {
					slog.Error("Could not send SIGTERM.", "error", err)
				}
				if err := prevCmd.Wait(); err != nil {
					slog.Error("Command failed.", "error", err)
				}
			}

			cmd, err := w.runCmd(fname)
			if err != nil {
				slog.Error("Command failed.", "error", err)
				goto outer
			}
			if w.mode == modeService {
				prevCmd = cmd
				break
			} else {
				if err := cmd.Wait(); err != nil {
					slog.Error("Command failed.", "error", err)
				}
			}
		}
		goto outer
	}
}
