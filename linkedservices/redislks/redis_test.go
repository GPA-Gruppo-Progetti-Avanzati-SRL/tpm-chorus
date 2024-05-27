package redislks_test

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"tpm-chorus/linkedservices/redislks"
)

func TestNewInstanceWithConfig(t *testing.T) {

	cfg := redislks.Config{
		Addr: "localhost:6379",
	}

	lks, err := redislks.NewInstanceWithConfig(&cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup

	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go putMessage(t, lks, fmt.Sprintf("MSG-%2d", i), fmt.Sprintf("MSG-%2d-Value", i), &wg)
	}

	t.Log("Waiting for goroutines  put to finish...")
	wg.Wait()

	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go getMessage(t, lks, fmt.Sprintf("MSG-%2d", i), &wg)
	}

	t.Log("Waiting for goroutines  put to finish...")
	wg.Wait()

}

func putMessage(t *testing.T, lks *redislks.LinkedService, k, v string, wg *sync.WaitGroup) {
	defer wg.Done()
	err := lks.Set(context.Background(), k, v)
	if err != nil {
		t.Error(err)
	}

	t.Logf("cached %s --> %s", k, v)
}

func getMessage(t *testing.T, lks *redislks.LinkedService, k string, wg *sync.WaitGroup) {
	defer wg.Done()
	v, err := lks.Get(context.Background(), k)
	if err != nil {
		t.Error(err)
	}

	if v == nil {
		t.Errorf("no value found for %s --> %v", k, v)
	} else {
		t.Logf("retrieved val %s --> %v", k, v)
	}
}
