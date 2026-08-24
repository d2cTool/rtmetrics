package common

// Assign копирует *src в dest, если поле задано в JSON (указатель не nil).
func Assign[T any](dest *T, src *T) {
	if src != nil {
		*dest = *src
	}
}

// AssignFunc копирует преобразованное *src в dest. src == nil — поле не задано.
func AssignFunc[S, D any](dest *D, src *S, conv func(S) (D, error)) error {
	if src == nil {
		return nil
	}
	v, err := conv(*src)
	if err != nil {
		return err
	}
	*dest = v
	return nil
}
