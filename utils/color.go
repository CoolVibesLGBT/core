package utils

import "crypto/md5"

// RankGradient struct
type RankGradient struct {
	Colors    [2]string
	TextColor string
}

var rainbowRankGradients = []RankGradient{
	{Colors: [2]string{"#FF3B30", "#FF6B3B"}, TextColor: "#fff"},
	{Colors: [2]string{"#FF9500", "#FFD60A"}, TextColor: "#fff"},
	{Colors: [2]string{"#FFD60A", "#34C759"}, TextColor: "#fff"},
	{Colors: [2]string{"#34C759", "#32D74B"}, TextColor: "#fff"},
	{Colors: [2]string{"#007AFF", "#5AC8FA"}, TextColor: "#fff"},
	{Colors: [2]string{"#5856D6", "#5E5CE6"}, TextColor: "#fff"},
	{Colors: [2]string{"#AF52DE", "#FF2D55"}, TextColor: "#fff"},
}

func hashStringToNumber(str string) int {
	hash := 0
	for _, c := range str {
		hash = (hash << 5) - hash + int(c)
		hash &= 0xFFFFFFFF
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

func GetRankGradient(tag interface{}) RankGradient {
	switch v := tag.(type) {
	case int:
		index := (v - 1) % len(rainbowRankGradients)
		if index < 0 {
			index = 0
		}
		return rainbowRankGradients[index]

	case string:
		hash := hashStringToNumber(v)
		index := hash % len(rainbowRankGradients)
		return rainbowRankGradients[index]

	default:
		return rainbowRankGradients[0]
	}
}

func NameToColor(name string) int {
	hash := md5.Sum([]byte(name))
	color := int(hash[0])<<16 | int(hash[1])<<8 | int(hash[2])
	return color
}
