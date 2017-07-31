package sys

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"git.wolaidai.com/DevOps/eye-cron/common/util"

	"github.com/itchenyi/file"
)

func CmdOut(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

func CmdOutBytes(name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.Bytes(), err
}

func CmdOutNoLn(name string, arg ...string) (out string, err error) {
	out, err = CmdOut(name, arg...)
	if err != nil {
		return
	}

	return strings.TrimSpace(string(out)), nil
}

func CmdRunWithTimeout(cmd *exec.Cmd, timeout time.Duration) (error, bool) {
	var err error

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		log.Printf("timeout, process:%s will be killed", cmd.Path)

		go func() {
			<-done // allow goroutine to exit
		}()

		//IMPORTANT: cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} is necessary before cmd.Start()
		err = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if err != nil {
			log.Println("kill failed, error:", err)
		}

		return err, true
	case err = <-done:
		return err, false
	}
}

func ScriptRunWithTimeOut(fileData string, filePath string, timeout int32) (string, error) {
	if err := file.WriteFile(filePath, util.Str2bytes(fileData), 0); err != nil {
		return "", fmt.Errorf("failed to write script file: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
	defer cancel()
	defer os.Remove(filePath)

	cmd := exec.CommandContext(ctx, "bash", "-c", filePath)
	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", errors.New("Execution timed out")
	}

	if err != nil {
		return string(out), fmt.Errorf("Non-zero exit code: %s", err)
	}

	return string(out), nil
}
