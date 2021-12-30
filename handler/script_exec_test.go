package handler_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/fsutil"
	"github.com/stretchr/testify/assert"
)

func TestScriptHandler_Execute_shell(t *testing.T) {
	assert.True(t, fsutil.IsFile("testdata/demo_run_shell.sh"))

	// use args
	args := []string{"testdata/demo_run_shell.sh", "arg0", "arg1", "0", "1"}
	cmd := exec.Command("bash", args...)
	output, err := cmd.Output()
	dump.P(string(output), err)
	assert.NoError(t, err)
	assert.Contains(t, string(output), "脚本位置：testdata/demo_run_shell.sh")

	// use -c
	// run: bash -c handler/testdata/demo_run_shell.sh arg0 arg1 0 1 // need exec prem
	code := "testdata/demo_run_shell.sh arg0 arg1 0 1"
	cmd = exec.Command("bash", "-c", code)
	output, err = cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			dump.P(string(ee.Stderr))
		}
	}
	dump.P(cmd.String(), string(output), err.Error())
	assert.Error(t, err)

	// not use -c
	cmd = exec.Command("bash", code)
	output, err = cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			dump.P(string(ee.Stderr))
		}
	}
	dump.P(cmd.String(), string(output), err.Error())
	assert.Error(t, err)
}

func TestScriptHandler_Execute_shell2file(t *testing.T) {

}

func TestScriptHandler_Execute_python2(t *testing.T) {
	assert.True(t, fsutil.IsFile("testdata/test_run_python2.py"))

	_, err := exec.LookPath("python")
	if err != nil {
		return // not found python on system
	}

	// run: python testdata/test_run_python2.py arg0 arg1 0 1
	args := []string{"testdata/test_run_python2.py", "arg0", "arg1", "0", "1"}
	cmd := exec.Command("python", args...)
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	assert.NoError(t, err)

	cmd = exec.Command("python", args...)
	output, err := cmd.Output()
	dump.P(string(output), err)
	assert.NoError(t, err)

	assert.Contains(t, string(output), "脚本位置： testdata/test_run_python2.py")

	// error
	code := "testdata/test_run_python2.py arg0 arg1 0 1"
	cmd = exec.Command("python", code)
	_, err = cmd.Output()
	assert.Error(t, err)
}

func TestScriptHandler_Execute_python3(t *testing.T) {
	assert.True(t, fsutil.IsFile("testdata/test_run_python3.py"))

	_, err := exec.LookPath("python3")
	if err != nil {
		return // not found python3 on system
	}

	// run: python testdata/test_run_python3.py arg0 arg1 0 1
	args := []string{"testdata/test_run_python3.py", "arg0", "arg1", "0", "1"}
	cmd := exec.Command("python3", args...)
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	assert.NoError(t, err)

	cmd = exec.Command("python3", args...)
	output, err := cmd.Output()
	dump.P(string(output), err)
	assert.NoError(t, err)
	assert.Contains(t, string(output), "脚本位置： testdata/test_run_python3.py")

	code := "testdata/test_run_python3.py arg0 arg1 0 1"
	cmd = exec.Command("python3", code)
	_, err = cmd.Output()
	assert.Error(t, err)
}
