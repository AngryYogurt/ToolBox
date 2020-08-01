package model

type Product struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	Description   string         `json:"description"`
	Category      string         `json:"category"`
	CharacterType string         `json:"character_type"`
	Name          string         `json:"name"`
	MotionID      string         `json:"motion_id"`
	Details       *ProductDetail `json:"details"`
	Source        string         `json:"source"`
}

type ProductDetail struct {
	SupportsInplace    bool            `json:"supports_inplace"`
	Loopable           bool            `json:"loopable"`
	DefaultFrameLength int             `json:"default_frame_length"`
	Duration           float64         `json:"duration"`
	GmsHash            *GmsHash        `json:"gms_hash"`
	Motions            []*DetailMotion `json:"motions"`
}

type DetailMotion struct {
	SupportsInplace    bool     `json:"supports_inplace"`
	Loopable           bool     `json:"loopable"`
	DefaultFrameLength int      `json:"default_frame_length"`
	Duration           float64  `json:"duration"`
	Name               string   `json:"name"`
	GmsHash            *GmsHash `json:"gms_hash"`
}

type GmsHash struct {
	ModelID   int         `json:"model-id"`
	Mirror    bool        `json:"mirror"`
	Trim      []float64   `json:"trim"`
	Inplace   bool        `json:"inplace"`
	ArmSpace  int         `json:"arm-space"`
	Params    interface{} `json:"params"`
	Name      string      `json:"name,omitempty"`
	Overdrive int         `json:"overdrive"`
}
