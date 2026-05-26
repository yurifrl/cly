package notify

import _ "embed"

//go:embed icons/failing.png
var iconFailing []byte

//go:embed icons/recovered.png
var iconRecovered []byte

//go:embed icons/gaveup.png
var iconGaveUp []byte

func iconBytes(l Level) []byte {
	switch l {
	case LevelFailing:
		return iconFailing
	case LevelRecovered:
		return iconRecovered
	case LevelGaveUp:
		return iconGaveUp
	}
	return nil
}
