package searchidx

// //////////////////////////////////////////////////////////////////
// //
// //                       Search Index
// //
// //////////////////////////////////////////////////////////////////

// // SearchIdx store search index version
// type SearchIdx interface {
// 	GetSiVersion() uint32
// }

// // SiVersionS is structure used to store search index version
// type SiVersionS struct {
// 	SiVer uint32
// }

// // GetSiVersion retrieves the search index version number
// func (h *SiVersionS) GetSiVersion() uint32 {
// 	return h.SiVer
// }

// // Deserialize deserializes the data into version number object
// func (h *SiVersionS) Deserialize(data []byte) (*SiVersionS, error) {
// 	err := json.Unmarshal(data, h)
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}
// 	return h, nil
// }

// // DeserializeSiVersion deserializes the data into version number object
// func DeserializeSiVersion(data []byte) (*SiVersionS, error) {
// 	h := &SiVersionS{}
// 	return h.Deserialize(data)
// }
