// bake-font bakes a variable TTF at a chosen wght default by patching
// the fvar table and the OS/2 usWeightClass field in the binary.
//
// Usage:
//
//	go run ./scripts/bake-font input.wght.ttf output.ttf [newDefaultWeight]
//
// The default newDefault is 450 (between Regular 400 and Medium 500),
// which makes CJK glyphs in CascadiaNextSC noticeably more readable.
//
// The font is loaded as a variable font in Fyne's text shaper; because
// Fyne's loader does not call SetVariations, glyph outlines are
// interpolated at the fvar axis default value. Patching that default
// in the binary is the only place we can influence the rendered weight
// from the Go side without forking Fyne's font loader.
//
// Only the wght axis is touched. Other axes (if any) keep their
// original default.
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: bake-font input.ttf output.ttf [newDefaultWeight]")
		os.Exit(2)
	}
	inPath := os.Args[1]
	outPath := os.Args[2]
	newDefault := uint16(450)
	if len(os.Args) >= 4 {
		v, err := strconv.ParseUint(os.Args[3], 10, 16)
		if err != nil {
			fmt.Fprintln(os.Stderr, "bad weight:", err)
			os.Exit(2)
		}
		newDefault = uint16(v)
	}

	data, err := os.ReadFile(inPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}
	out := make([]byte, len(data))
	copy(out, data)

	// Walk the TTF table directory.
	numTables := int(binary.BigEndian.Uint16(data[4:6]))
	const wghtTag = "wght"
	for i := 0; i < numTables; i++ {
		off := 12 + i*16
		tag := string(data[off : off+4])
		tableOff := binary.BigEndian.Uint32(data[off+8 : off+12])
		tableLen := binary.BigEndian.Uint32(data[off+12 : off+16])
		switch tag {
		case "fvar":
			fvar := out[tableOff : tableOff+tableLen]
			// fvar header:
			//   uint16 majorVersion, uint16 minorVersion,
			//   uint16 axesArrayOffset, uint16 reserved,
			//   uint16 axisCount, uint16 axisSize,
			//   uint16 instanceCount, uint16 instanceSize
			axisCount := int(binary.BigEndian.Uint16(fvar[8:10]))
			axisSize := int(binary.BigEndian.Uint16(fvar[10:12]))
			axesOff := int(binary.BigEndian.Uint16(fvar[4:6]))
			// Each VariationAxisRecord:
			//   Tag(4) minValue(Fixed) defaultValue(Fixed) maxValue(Fixed) flags(U16) axisNameID(U16)
			// Fixed is 16.16 (signed).
			for j := 0; j < axisCount; j++ {
				ax := fvar[axesOff+j*axisSize : axesOff+(j+1)*axisSize]
				if len(ax) < 20 {
					continue
				}
				tag := string(ax[0:4])
				if tag == wghtTag {
					old := binary.BigEndian.Uint32(ax[8:12])
					// Replace default with the user-requested weight (Fixed 16.16).
					binary.BigEndian.PutUint32(ax[8:12], uint32(newDefault)<<16)
					fmt.Printf("patched fvar[%d] %s default %.0f -> %d\n", j, tag, float32(old)/65536, newDefault)
				}
			}
		case "OS/2":
			os2 := out[tableOff : tableOff+tableLen]
			if len(os2) >= 6 {
				old := binary.BigEndian.Uint16(os2[4:6])
				binary.BigEndian.PutUint16(os2[4:6], newDefault)
				fmt.Printf("patched OS/2 usWeightClass %d -> %d\n", old, newDefault)
			}
		}
	}

	if err := os.WriteFile(outPath, out, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d bytes, default wght=%d)\n", outPath, len(out), newDefault)
}
