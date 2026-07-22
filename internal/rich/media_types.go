package rich

// PhotoSize описывает один размер фотографии во входящем RichBlockPhoto.
type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id,omitempty"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int    `json:"file_size,omitempty"`
}

// Video описывает видео во входящем RichBlockVideo.
type Video struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id,omitempty"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Duration     int    `json:"duration"`
	FileSize     int    `json:"file_size,omitempty"`
}

// Animation описывает анимацию во входящем RichBlockAnimation.
type Animation struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id,omitempty"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Duration     int    `json:"duration"`
	FileSize     int    `json:"file_size,omitempty"`
}

// Audio описывает аудио во входящем RichBlockAudio.
type Audio struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id,omitempty"`
	Duration     int    `json:"duration"`
	Performer    string `json:"performer,omitempty"`
	Title        string `json:"title,omitempty"`
	FileSize     int    `json:"file_size,omitempty"`
}

// Voice описывает голосовое сообщение во входящем RichBlockVoiceNote.
type Voice struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id,omitempty"`
	Duration     int    `json:"duration"`
	FileSize     int    `json:"file_size,omitempty"`
}

// Location описывает геопозицию блока RichBlockMap.
type Location struct {
	Longitude            float64 `json:"longitude"`
	Latitude             float64 `json:"latitude"`
	HorizontalAccuracy   float64 `json:"horizontal_accuracy,omitempty"`
	LivePeriod           int     `json:"live_period,omitempty"`
	Heading              int     `json:"heading,omitempty"`
	ProximityAlertRadius int     `json:"proximity_alert_radius,omitempty"`
}

// bestPhotoSize возвращает PhotoSize с наибольшим доступным file_id по размеру.
func bestPhotoSize(photos []PhotoSize) (PhotoSize, bool) {
	if len(photos) == 0 {
		return PhotoSize{}, false
	}
	best := photos[0]
	bestScore := photoScore(best)
	for _, photo := range photos[1:] {
		score := photoScore(photo)
		if score > bestScore {
			best = photo
			bestScore = score
		}
	}
	return best, best.FileID != ""
}

func photoScore(photo PhotoSize) int {
	if photo.FileSize > 0 {
		return photo.FileSize
	}
	return photo.Width * photo.Height
}
