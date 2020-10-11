package db

type RowData map[string]interface{}

func (data *RowData) Get(name string) (interface{}, bool) {
	v, ok := (*data)[name]
	return v, ok
}

func (data *RowData) Set(name string, v interface{}) {
	(*data)[name] = v
}

func (data *RowData) Raw() *map[string]interface{} {
	return (*map[string]interface{})(data)
}

func (data *RowData) Len() int {
	return len(*data.Raw())
}
