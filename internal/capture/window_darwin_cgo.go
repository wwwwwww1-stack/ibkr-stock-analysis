//go:build darwin && cgo

package capture

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>

char* copyCFStringUTF8(CFStringRef value) {
	if (value == NULL) {
		return NULL;
	}
	CFIndex length = CFStringGetLength(value);
	CFIndex maxSize = CFStringGetMaximumSizeForEncoding(length, kCFStringEncodingUTF8) + 1;
	char *buffer = (char*)malloc(maxSize);
	if (buffer == NULL) {
		return NULL;
	}
	if (!CFStringGetCString(value, buffer, maxSize, kCFStringEncodingUTF8)) {
		free(buffer);
		return NULL;
	}
	return buffer;
}

char* copyDictString(CFDictionaryRef dict, const void *key) {
	CFTypeRef value = CFDictionaryGetValue(dict, key);
	if (value == NULL || CFGetTypeID(value) != CFStringGetTypeID()) {
		return NULL;
	}
	return copyCFStringUTF8((CFStringRef)value);
}

int dictInt(CFDictionaryRef dict, const void *key) {
	CFTypeRef value = CFDictionaryGetValue(dict, key);
	if (value == NULL || CFGetTypeID(value) != CFNumberGetTypeID()) {
		return 0;
	}
	int out = 0;
	CFNumberGetValue((CFNumberRef)value, kCFNumberIntType, &out);
	return out;
}

CFArrayRef copyVisibleWindowInfo() {
	return CGWindowListCopyWindowInfo(kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements, kCGNullWindowID);
}

CFIndex windowInfoCount(CFArrayRef windows) {
	return windows == NULL ? 0 : CFArrayGetCount(windows);
}

CFDictionaryRef windowInfoAt(CFArrayRef windows, CFIndex index) {
	return (CFDictionaryRef)CFArrayGetValueAtIndex(windows, index);
}

int windowNumber(CFDictionaryRef window) {
	return dictInt(window, kCGWindowNumber);
}

int windowLayer(CFDictionaryRef window) {
	return dictInt(window, kCGWindowLayer);
}

char* windowOwnerName(CFDictionaryRef window) {
	return copyDictString(window, kCGWindowOwnerName);
}

char* windowTitle(CFDictionaryRef window) {
	return copyDictString(window, kCGWindowName);
}

void releaseCFType(CFTypeRef value) {
	if (value != NULL) {
		CFRelease(value);
	}
}
*/
import "C"

import (
	"context"
	"fmt"
	"unsafe"
)

func platformListWindows(ctx context.Context) ([]WindowInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	windowsRef := C.copyVisibleWindowInfo()
	if unsafe.Pointer(windowsRef) == nil {
		return nil, fmt.Errorf("list chart windows: CoreGraphics returned no windows")
	}
	defer C.releaseCFType(C.CFTypeRef(windowsRef))

	count := int(C.windowInfoCount(windowsRef))
	windows := make([]WindowInfo, 0, count)
	for i := 0; i < count; i++ {
		windowRef := C.windowInfoAt(windowsRef, C.CFIndex(i))
		if unsafe.Pointer(windowRef) == nil || int(C.windowLayer(windowRef)) != 0 {
			continue
		}
		id := int(C.windowNumber(windowRef))
		owner := cString(C.windowOwnerName(windowRef))
		title := cString(C.windowTitle(windowRef))
		windows = append(windows, WindowInfo{ID: id, AppName: owner, Title: title})
	}
	return sanitizeWindows(windows, nil)
}

func cString(value *C.char) string {
	if value == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(value))
	return C.GoString(value)
}
