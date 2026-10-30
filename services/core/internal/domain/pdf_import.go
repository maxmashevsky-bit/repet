package domain

import "time"

type PDFImportInput struct {
	RequestID    string `json:"request_id"`
	Filename     string `json:"filename"`
	DataBase64   string `json:"data_base64"`
	RightsStatus string `json:"rights_status"`
	License      string `json:"license"`
	SourceLabel  string `json:"source_label"`
}
type PDFAsset struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	PageNumber int    `json:"page_number"`
	Ordinal    int    `json:"ordinal"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Bytes      int64  `json:"bytes"`
}
type PDFPage struct {
	Number        int        `json:"number"`
	OriginalText  string     `json:"original_text"`
	ImageID       string     `json:"image_id"`
	OCRRequired   bool       `json:"ocr_required"`
	CorrectedText *string    `json:"corrected_text"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
}
type PDFImport struct {
	LessonIDs    []string   `json:"lesson_ids"`
	ID           string     `json:"id"`
	Filename     string     `json:"filename"`
	SHA256       string     `json:"sha256"`
	Bytes        int64      `json:"source_bytes"`
	RightsStatus string     `json:"rights_status"`
	License      string     `json:"license"`
	SourceLabel  string     `json:"source_label"`
	Status       string     `json:"status"`
	Warnings     []string   `json:"warnings"`
	Failure      string     `json:"failure"`
	CreatedAt    time.Time  `json:"created_at"`
	Pages        []PDFPage  `json:"pages"`
	Assets       []PDFAsset `json:"assets"`
}
