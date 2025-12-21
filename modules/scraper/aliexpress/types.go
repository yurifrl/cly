package aliexpress

import "time"

// ProductData represents complete AliExpress product data
type ProductData struct {
	ProductID   int64        `json:"productId"`
	Title       string       `json:"title"`
	CategoryID  int64        `json:"categoryId,omitempty"`
	Orders      string       `json:"orders,omitempty"`
	Images      []string     `json:"images"`
	Variants    VariantsData `json:"variants"`
	Reviews     []Review     `json:"reviews"`
	Ratings     RatingsData  `json:"ratings"`
	StoreInfo   StoreInfo    `json:"storeInfo"`
	Specs       []Spec       `json:"specs"`
	Shipping    []Shipping   `json:"shipping"`
	Description string       `json:"description"`
	Currency    Currency     `json:"currencyInfo"`
	Quantity    Quantity     `json:"quantity"`

	// Metadata
	ScrapedAt time.Time `json:"scrapedAt"`
	SourceURL string    `json:"sourceUrl"`
}

// VariantsData represents product variants and pricing
type VariantsData struct {
	Options []VariantOption `json:"options"`
	Prices  []SKUPrice      `json:"prices"`
}

// VariantOption represents a variant option (color, size, etc.)
type VariantOption struct {
	ID     int64          `json:"id"`
	Name   string         `json:"name"`
	Values []VariantValue `json:"values"`
}

// VariantValue represents a specific variant value
type VariantValue struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Image       string `json:"image,omitempty"`
}

// SKUPrice represents pricing for a specific SKU
type SKUPrice struct {
	SKUID             int64  `json:"skuId"`
	OptionValueIDs    string `json:"optionValueIds"`
	AvailableQuantity int    `json:"availableQuantity"`
	OriginalPrice     Price  `json:"originalPrice"`
	SalePrice         Price  `json:"salePrice"`
}

// Price represents a price with currency
type Price struct {
	Currency       string  `json:"currency"`
	FormattedAmount string  `json:"formatedAmount"`
	Value          float64 `json:"value"`
}

// Review represents a product review
type Review struct {
	ID          string    `json:"id,omitempty"`
	Anonymous   bool      `json:"anonymous"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	Gender      string    `json:"gender"`
	Country     string    `json:"country"`
	Rating      int       `json:"rating"`
	Info        string    `json:"info,omitempty"`
	Date        string    `json:"date"`
	Content     string    `json:"content"`
	Photos      []string  `json:"photos"`
	Thumbnails  []string  `json:"thumbnails"`
}

// RatingsData represents product ratings breakdown
type RatingsData struct {
	TotalStar      int     `json:"totalStar"`
	AverageStar    string  `json:"averageStar"`
	TotalCount     int64   `json:"totalStartCount"`
	FiveStarCount  int64   `json:"fiveStarCount"`
	FourStarCount  int64   `json:"fourStarCount"`
	ThreeStarCount int64   `json:"threeStarCount"`
	TwoStarCount   int64   `json:"twoStarCount"`
	OneStarCount   int64   `json:"oneStarCount"`
}

// StoreInfo represents store information
type StoreInfo struct {
	Name              string `json:"name"`
	Logo              string `json:"logo"`
	CompanyID         int64  `json:"companyId"`
	StoreNumber       int64  `json:"storeNumber"`
	IsTopRated        bool   `json:"isTopRated"`
	HasPayPalAccount  bool   `json:"hasPayPalAccount"`
	RatingCount       int    `json:"ratingCount"`
	Rating            string `json:"rating"`
}

// Spec represents a product specification
type Spec struct {
	AttrName  string `json:"attrName"`
	AttrValue string `json:"attrValue"`
}

// Shipping represents shipping information
type Shipping struct {
	DeliveryProviderName string       `json:"deliveryProviderName"`
	Tracking             string       `json:"tracking"`
	Provider             string       `json:"provider"`
	Company              string       `json:"company"`
	DeliveryInfo         DeliveryInfo `json:"deliveryInfo"`
	ShippingInfo         ShippingInfo `json:"shippingInfo"`
	WarehouseType        string       `json:"warehouseType"`
}

// DeliveryInfo represents delivery time information
type DeliveryInfo struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// ShippingInfo represents shipping details
type ShippingInfo struct {
	From            string  `json:"from"`
	FromCode        string  `json:"fromCode"`
	To              string  `json:"to"`
	ToCode          string  `json:"toCode"`
	Fees            string  `json:"fees"`
	DisplayAmount   float64 `json:"displayAmount"`
	DisplayCurrency string  `json:"displayCurrency"`
}

// Currency represents currency information
type Currency struct {
	BaseCurrencyCode   string  `json:"baseCurrencyCode"`
	EnableTransaction  bool    `json:"enableTransaction"`
	CurrencySymbol     string  `json:"currencySymbol"`
	SymbolFront        bool    `json:"symbolFront"`
	CurrencyRate       float64 `json:"currencyRate"`
	BaseSymbolFront    bool    `json:"baseSymbolFront"`
	CurrencyCode       string  `json:"currencyCode"`
	BaseCurrencySymbol string  `json:"baseCurrencySymbol"`
	MultiCurrency      bool    `json:"multiCurrency"`
}

// Quantity represents product quantity information
type Quantity struct {
	Total     int `json:"total"`
	Available int `json:"available"`
}
