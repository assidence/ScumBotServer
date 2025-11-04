package Public

type AchievementInterfaceStruct struct {
	AchievementReward map[string][]string
}

var AchievementInterface *AchievementInterfaceStruct

func init() {
	AchievementInterface = &AchievementInterfaceStruct{
		make(map[string][]string),
	}
}
