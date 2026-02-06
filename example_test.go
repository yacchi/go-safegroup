package safegroup_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/yacchi/go-safegroup"
)

func ExampleWithContext() {
	group, _ := safegroup.WithContext(context.Background())

	group.GoLabel("tenant=A/job=1", func(context.Context) error {
		return errors.New("regular error")
	})
	group.GoLabel("tenant=A/job=2", func(context.Context) error {
		panic("unexpected")
	})

	if err := group.Wait(); err != nil {
		fmt.Println(safegroup.IsPanic(err))
		fmt.Println(len(safegroup.AllPanics(err)) > 0)
	}

	// Output:
	// true
	// true
}
