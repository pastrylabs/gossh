// Code generated by "stringer -type=Fact"; DO NOT EDIT.

package gossh

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[OS-0]
	_ = x[OSFamily-1]
	_ = x[OSVersion-2]
}

const _Fact_name = "OSOSFamilyOSVersion"

var _Fact_index = [...]uint8{0, 2, 10, 19}

func (i Fact) String() string {
	if i < 0 || i >= Fact(len(_Fact_index)-1) {
		return "Fact(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Fact_name[_Fact_index[i]:_Fact_index[i+1]]
}