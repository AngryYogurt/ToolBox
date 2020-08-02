package constant

const (
	AnimationListURL   = "https://www.mixamo.com/api/v1/products?page=%d&limit=96&order=&type=Motion%%2CMotionPack&query="
	GetProductURL      = "https://www.mixamo.com/api/v1/products/%s?similar=0&character_id=%s"
	ExportAnimationURL = "https://www.mixamo.com/api/v1/animations/export"
	MonitorURL         = "https://www.mixamo.com/api/v1/characters/%s/monitor"
)

const (
	DataDir           = "./mixamo/data"
	AnimationListFile = "animation_list.json"
	AnimationListFile2 = "animation_list2.json"
	AnimationListFile3 = "animation_list3.json"
	AnimationListFile4 = "animation_list4.json"
	AnimationListFile5 = "animation_list5.json"
	AllAnimationListFile = "all_animation_list.json"

	WithSkin       = "false"
	MeshMotionpack = "no-character" // or "original" or "t-pose"
	Fps            = "30"
)
