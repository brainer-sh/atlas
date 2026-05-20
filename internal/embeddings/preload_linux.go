//go:build with_embeddings && linux

package embeddings

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>

static int tryDlopen(const char* path) {
	void *h = dlopen(path, RTLD_GLOBAL | RTLD_LAZY);
	return h != NULL ? 1 : 0;
}
*/
import "C"

import (
	"log/slog"
	"os/exec"
	"strings"
	"unsafe"
)

// preloadLibstdcxx opens libstdc++.so.6 into the global symbol namespace before
// ORT is loaded. This prevents "libstdc++.so.6: cannot open shared object file"
// on NixOS and other distros where libstdc++ is not in the default search path.
func preloadLibstdcxx() {
	// Most distros: libstdc++ is in the default LD path.
	name := C.CString("libstdc++.so.6")
	defer C.free(unsafe.Pointer(name))
	if C.tryDlopen(name) == 1 {
		return
	}

	// Fallback: ask gcc for the full path.
	out, err := exec.Command("gcc", "-print-file-name=libstdc++.so.6").Output()
	if err == nil {
		p := strings.TrimSpace(string(out))
		// gcc returns its argument unchanged when the file is not found.
		if p != "libstdc++.so.6" && p != "" {
			cp := C.CString(p)
			defer C.free(unsafe.Pointer(cp))
			if C.tryDlopen(cp) == 1 {
				slog.Debug("preloaded libstdc++ via gcc", "path", p)
				return
			}
		}
	}

	// Not fatal: ORT may still load on distros that embed libstdc++ statically.
	slog.Debug("could not preload libstdc++.so.6; ORT may fail on NixOS")
}
