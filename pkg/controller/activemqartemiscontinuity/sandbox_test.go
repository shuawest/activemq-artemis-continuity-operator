package activemqartemiscontinuity

import (
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var testlog = logf.Log.WithName("test_controller_activemqartemiscontinuity")

func TestSuccess1(t *testing.T) {
	pflag.CommandLine.AddFlagSet(zap.FlagSet())
	pflag.Parse()
	logf.SetLogger(zap.Logger())

	result, err := runApp("success")

	log.Info("TestSuccess1", "result", result, "err", err)

	assert.NoError(t, err)
	assert.Equal(t, true, result)
}

func TestFail1(t *testing.T) {
	result, err := runApp("fail")

	log.Info("TestFail1", "result", result, "err", err)

	assert.NoError(t, err)
	assert.Equal(t, false, result)
}

func TestError1(t *testing.T) {
	result, err := runApp("error")

	log.Info("TestError1", "result", result, "err", err)

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "hit error")
	}
	assert.Equal(t, false, result)
}

func TestError2(t *testing.T) {
	result, err := runApp("unknown")

	log.Info("TestError2", "result", result, "err", err)

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "hit default")
	}
	assert.Equal(t, false, result)
}

func runApp(whatToDo string) (bool, error) {
	switch whatToDo {
	case "success":
		return true, nil
	case "fail":
		return false, nil
	case "error":
		var myerr error = &MyError{
			time.Now(),
			"hit error",
		}
		return false, myerr
	default:
		var myerr error = &MyError{
			time.Now(),
			"hit default",
		}
		return false, myerr
	}
}
