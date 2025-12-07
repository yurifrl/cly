package skin

//go:generate go tool go-enum --file $GOFILE --noprefix --marshal --names

var skin *Skin

func SetSkin(s Skin) {
	skin = &s
}

func GetSkin() Skin {
	return *skin
}

func GetSkinAndEnv() (Skin, string) {
	skin := GetSkin()
	switch skin {
	case Betdev:
		return Betdev, "staging"
	case Betnacional:
		return Betnacional, "production"
	case Mrjackbets:
		return Mrjackbets, "production"
	case Mundialbet:
		return Mundialbet, "homol"
	default:
		return skin, ""
	}
}

// ENUM(
//
//		betdev
//		betnacional
//		mrjackbets
//	 mundialbet
//
// )
type Skin uint8
