package control

import (
	"errors"
	"os"
	"syscall"
	"testing"
)

func TestIsSocketChmodUnsupported(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "einval", err: syscall.EINVAL, want: true},
		{name: "eopnotsupp", err: syscall.EOPNOTSUPP, want: true},
		{name: "eacces", err: syscall.EACCES, want: false},
		{
			name: "path_einval",
			err:  &os.PathError{Op: "chmod", Path: "/run/x.sock", Err: syscall.EINVAL},
			want: true,
		},
		{
			name: "wrapped_einval",
			err:  errors.Join(errors.New("chmod control socket"), &os.PathError{Op: "chmod", Path: "x", Err: syscall.EINVAL}),
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSocketChmodUnsupported(tc.err); got != tc.want {
				t.Fatalf("got %v want %v for %v", got, tc.want, tc.err)
			}
		})
	}
}
