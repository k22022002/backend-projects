package component

var Descriptions = map[int]string{}

func SetDescription(id int, desc string) {
	Descriptions[id] = desc
}

func GetDescription(id int) string {
	if desc, ok := Descriptions[id]; ok {
		return desc
	}
	return "" // hoặc trả về "Unknown description"
}
