package gslides

// Minimal hand-rolled Google Slides API v1 request structs — only the
// requests and fields this emitter sets. Field names match the REST JSON
// wire format exactly; goldens in testdata pin the encoding. Using our own
// structs keeps the module dependency-free and makes the emitter a pure,
// testable ir.Deck -> []Request function.

// Request is the batchUpdate request union; exactly one field is set.
type Request struct {
	CreateSlide                 *CreateSlideRequest                 `json:"createSlide,omitempty"`
	DeleteObject                *DeleteObjectRequest                `json:"deleteObject,omitempty"`
	UpdatePageProperties        *UpdatePagePropertiesRequest        `json:"updatePageProperties,omitempty"`
	CreateShape                 *CreateShapeRequest                 `json:"createShape,omitempty"`
	CreateLine                  *CreateLineRequest                  `json:"createLine,omitempty"`
	CreateTable                 *CreateTableRequest                 `json:"createTable,omitempty"`
	InsertText                  *InsertTextRequest                  `json:"insertText,omitempty"`
	UpdateTextStyle             *UpdateTextStyleRequest             `json:"updateTextStyle,omitempty"`
	UpdateParagraphStyle        *UpdateParagraphStyleRequest        `json:"updateParagraphStyle,omitempty"`
	UpdateShapeProperties       *UpdateShapePropertiesRequest       `json:"updateShapeProperties,omitempty"`
	UpdateLineProperties        *UpdateLinePropertiesRequest        `json:"updateLineProperties,omitempty"`
	UpdateTableColumnProperties *UpdateTableColumnPropertiesRequest `json:"updateTableColumnProperties,omitempty"`
	UpdateTableRowProperties    *UpdateTableRowPropertiesRequest    `json:"updateTableRowProperties,omitempty"`
	UpdateTableCellProperties   *UpdateTableCellPropertiesRequest   `json:"updateTableCellProperties,omitempty"`
	UpdateTableBorderProperties *UpdateTableBorderPropertiesRequest `json:"updateTableBorderProperties,omitempty"`
	CreateParagraphBullets      *CreateParagraphBulletsRequest      `json:"createParagraphBullets,omitempty"`
}

// CreateParagraphBulletsRequest turns paragraphs into list items. Leading
// tab characters in each paragraph set the nesting level and are consumed
// when the request executes.
type CreateParagraphBulletsRequest struct {
	ObjectID     string             `json:"objectId"`
	CellLocation *TableCellLocation `json:"cellLocation,omitempty"`
	TextRange    *Range             `json:"textRange"`
	BulletPreset string             `json:"bulletPreset"`
}

type CreateSlideRequest struct {
	ObjectID             string           `json:"objectId,omitempty"`
	SlideLayoutReference *LayoutReference `json:"slideLayoutReference,omitempty"`
}

type LayoutReference struct {
	PredefinedLayout string `json:"predefinedLayout,omitempty"`
}

type DeleteObjectRequest struct {
	ObjectID string `json:"objectId"`
}

type UpdatePagePropertiesRequest struct {
	ObjectID       string          `json:"objectId"`
	PageProperties *PageProperties `json:"pageProperties"`
	Fields         string          `json:"fields"`
}

type PageProperties struct {
	PageBackgroundFill *PageBackgroundFill `json:"pageBackgroundFill,omitempty"`
}

type PageBackgroundFill struct {
	SolidFill *SolidFill `json:"solidFill,omitempty"`
}

type CreateShapeRequest struct {
	ObjectID          string                 `json:"objectId,omitempty"`
	ShapeType         string                 `json:"shapeType"`
	ElementProperties *PageElementProperties `json:"elementProperties"`
}

type CreateLineRequest struct {
	ObjectID          string                 `json:"objectId,omitempty"`
	Category          string                 `json:"category,omitempty"`
	ElementProperties *PageElementProperties `json:"elementProperties"`
}

type CreateTableRequest struct {
	ObjectID          string                 `json:"objectId,omitempty"`
	ElementProperties *PageElementProperties `json:"elementProperties"`
	Rows              int                    `json:"rows"`
	Columns           int                    `json:"columns"`
}

type PageElementProperties struct {
	PageObjectID string           `json:"pageObjectId"`
	Size         *Size            `json:"size,omitempty"`
	Transform    *AffineTransform `json:"transform,omitempty"`
}

type Size struct {
	Width  Dimension `json:"width"`
	Height Dimension `json:"height"`
}

type Dimension struct {
	Magnitude float64 `json:"magnitude"`
	Unit      string  `json:"unit"`
}

// AffineTransform maps local to page coordinates:
// x' = scaleX*x + shearX*y + translateX.
type AffineTransform struct {
	ScaleX     float64 `json:"scaleX"`
	ScaleY     float64 `json:"scaleY"`
	ShearX     float64 `json:"shearX,omitempty"`
	ShearY     float64 `json:"shearY,omitempty"`
	TranslateX float64 `json:"translateX"`
	TranslateY float64 `json:"translateY"`
	Unit       string  `json:"unit"`
}

type InsertTextRequest struct {
	ObjectID       string             `json:"objectId"`
	CellLocation   *TableCellLocation `json:"cellLocation,omitempty"`
	Text           string             `json:"text"`
	InsertionIndex int                `json:"insertionIndex"`
}

type TableCellLocation struct {
	RowIndex    int `json:"rowIndex"`
	ColumnIndex int `json:"columnIndex"`
}

type UpdateTextStyleRequest struct {
	ObjectID     string             `json:"objectId"`
	CellLocation *TableCellLocation `json:"cellLocation,omitempty"`
	Style        *TextStyle         `json:"style"`
	TextRange    *Range             `json:"textRange"`
	Fields       string             `json:"fields"`
}

type TextStyle struct {
	Bold            *bool          `json:"bold,omitempty"`
	Italic          *bool          `json:"italic,omitempty"`
	Underline       *bool          `json:"underline,omitempty"`
	Strikethrough   *bool          `json:"strikethrough,omitempty"`
	FontSize        *Dimension     `json:"fontSize,omitempty"`
	FontFamily      string         `json:"fontFamily,omitempty"`
	ForegroundColor *OptionalColor `json:"foregroundColor,omitempty"`
	Link            *Link          `json:"link,omitempty"`
}

type Link struct {
	URL string `json:"url,omitempty"`
}

type UpdateParagraphStyleRequest struct {
	ObjectID     string             `json:"objectId"`
	CellLocation *TableCellLocation `json:"cellLocation,omitempty"`
	Style        *ParagraphStyle    `json:"style"`
	TextRange    *Range             `json:"textRange"`
	Fields       string             `json:"fields"`
}

type ParagraphStyle struct {
	Alignment   string  `json:"alignment,omitempty"`
	LineSpacing float64 `json:"lineSpacing,omitempty"` // percent, 100 = single
}

// Range is a text range in UTF-16 code units.
type Range struct {
	Type       string `json:"type"`
	StartIndex *int   `json:"startIndex,omitempty"`
	EndIndex   *int   `json:"endIndex,omitempty"`
}

type UpdateShapePropertiesRequest struct {
	ObjectID        string           `json:"objectId"`
	ShapeProperties *ShapeProperties `json:"shapeProperties"`
	Fields          string           `json:"fields"`
}

type ShapeProperties struct {
	ShapeBackgroundFill *ShapeBackgroundFill `json:"shapeBackgroundFill,omitempty"`
	Outline             *Outline             `json:"outline,omitempty"`
	ContentAlignment    string               `json:"contentAlignment,omitempty"`
	Autofit             *Autofit             `json:"autofit,omitempty"`
}

type ShapeBackgroundFill struct {
	PropertyState string     `json:"propertyState,omitempty"`
	SolidFill     *SolidFill `json:"solidFill,omitempty"`
}

type Outline struct {
	PropertyState string       `json:"propertyState,omitempty"`
	OutlineFill   *OutlineFill `json:"outlineFill,omitempty"`
	Weight        *Dimension   `json:"weight,omitempty"`
	DashStyle     string       `json:"dashStyle,omitempty"`
}

type OutlineFill struct {
	SolidFill *SolidFill `json:"solidFill,omitempty"`
}

type Autofit struct {
	AutofitType string `json:"autofitType,omitempty"`
}

type UpdateLinePropertiesRequest struct {
	ObjectID       string          `json:"objectId"`
	LineProperties *LineProperties `json:"lineProperties"`
	Fields         string          `json:"fields"`
}

type LineProperties struct {
	LineFill   *LineFill  `json:"lineFill,omitempty"`
	Weight     *Dimension `json:"weight,omitempty"`
	DashStyle  string     `json:"dashStyle,omitempty"`
	StartArrow string     `json:"startArrow,omitempty"`
	EndArrow   string     `json:"endArrow,omitempty"`
}

type LineFill struct {
	SolidFill *SolidFill `json:"solidFill,omitempty"`
}

type UpdateTableColumnPropertiesRequest struct {
	ObjectID              string                 `json:"objectId"`
	ColumnIndices         []int                  `json:"columnIndices"`
	TableColumnProperties *TableColumnProperties `json:"tableColumnProperties"`
	Fields                string                 `json:"fields"`
}

type TableColumnProperties struct {
	ColumnWidth Dimension `json:"columnWidth"`
}

type UpdateTableRowPropertiesRequest struct {
	ObjectID           string              `json:"objectId"`
	RowIndices         []int               `json:"rowIndices"`
	TableRowProperties *TableRowProperties `json:"tableRowProperties"`
	Fields             string              `json:"fields"`
}

type TableRowProperties struct {
	MinRowHeight Dimension `json:"minRowHeight"`
}

type UpdateTableCellPropertiesRequest struct {
	ObjectID            string               `json:"objectId"`
	TableRange          *TableRange          `json:"tableRange"`
	TableCellProperties *TableCellProperties `json:"tableCellProperties"`
	Fields              string               `json:"fields"`
}

type TableRange struct {
	Location   TableCellLocation `json:"location"`
	RowSpan    int               `json:"rowSpan"`
	ColumnSpan int               `json:"columnSpan"`
}

type TableCellProperties struct {
	TableCellBackgroundFill *TableCellBackgroundFill `json:"tableCellBackgroundFill,omitempty"`
}

type TableCellBackgroundFill struct {
	SolidFill *SolidFill `json:"solidFill,omitempty"`
}

type UpdateTableBorderPropertiesRequest struct {
	ObjectID              string                 `json:"objectId"`
	BorderPosition        string                 `json:"borderPosition,omitempty"`
	TableBorderProperties *TableBorderProperties `json:"tableBorderProperties"`
	Fields                string                 `json:"fields"`
}

type TableBorderProperties struct {
	TableBorderFill *TableBorderFill `json:"tableBorderFill,omitempty"`
	Weight          *Dimension       `json:"weight,omitempty"`
}

type TableBorderFill struct {
	SolidFill *SolidFill `json:"solidFill,omitempty"`
}

type SolidFill struct {
	Color *OpaqueColor `json:"color,omitempty"`
	// Alpha is a pointer so an explicit 0 (fully transparent) survives
	// encoding; nil means the API default of 1.
	Alpha *float64 `json:"alpha,omitempty"`
}

type OptionalColor struct {
	OpaqueColor *OpaqueColor `json:"opaqueColor,omitempty"`
}

type OpaqueColor struct {
	RgbColor *RgbColor `json:"rgbColor,omitempty"`
}

type RgbColor struct {
	Red   float64 `json:"red,omitempty"`
	Green float64 `json:"green,omitempty"`
	Blue  float64 `json:"blue,omitempty"`
}
