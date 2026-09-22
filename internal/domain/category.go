package domain

// ValidCategories defines the allowed post categories.
// These are used for filtering posts on the mobile app and
// are enforced during post creation.
var ValidCategories = []string{
	"entertainment",
	"gaming",
	"sports",
	"news",
	"tech",
	"lifestyle",
	"education",
	"music",
	"art",
	"other",
}

// validCategorySet is a lookup map built from ValidCategories for O(1) checks.
var validCategorySet map[string]struct{}

func init() {
	validCategorySet = make(map[string]struct{}, len(ValidCategories))
	for _, c := range ValidCategories {
		validCategorySet[c] = struct{}{}
	}
}

// IsValidCategory returns true if the given name is in the predefined list.
func IsValidCategory(name string) bool {
	_, ok := validCategorySet[name]
	return ok
}
