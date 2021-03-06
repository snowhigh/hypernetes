/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// ************************************************************
// DO NOT EDIT.
// THIS FILE IS AUTO-GENERATED BY codecgen.
// ************************************************************

package v1

import (
	"errors"
	"fmt"
	codec1978 "github.com/ugorji/go/codec"
	pkg2_api "k8s.io/kubernetes/pkg/api"
	pkg1_unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	pkg3_types "k8s.io/kubernetes/pkg/types"
	"reflect"
	"runtime"
	time "time"
)

const (
	// ----- content types ----
	codecSelferC_UTF81234 = 1
	codecSelferC_RAW1234  = 0
	// ----- value types used ----
	codecSelferValueTypeArray1234 = 10
	codecSelferValueTypeMap1234   = 9
	// ----- containerStateValues ----
	codecSelfer_containerMapKey1234    = 2
	codecSelfer_containerMapValue1234  = 3
	codecSelfer_containerMapEnd1234    = 4
	codecSelfer_containerArrayElem1234 = 6
	codecSelfer_containerArrayEnd1234  = 7
)

var (
	codecSelferBitsize1234                         = uint8(reflect.TypeOf(uint(0)).Bits())
	codecSelferOnlyMapOrArrayEncodeToStructErr1234 = errors.New(`only encoded map or array can be decoded into a struct`)
)

type codecSelfer1234 struct{}

func init() {
	if codec1978.GenVersion != 5 {
		_, file, _, _ := runtime.Caller(0)
		err := fmt.Errorf("codecgen version mismatch: current: %v, need %v. Re-generate file: %v",
			5, codec1978.GenVersion, file)
		panic(err)
	}
	if false { // reference the types, but skip this branch at build/run time
		var v0 pkg2_api.ObjectMeta
		var v1 pkg1_unversioned.TypeMeta
		var v2 pkg3_types.UID
		var v3 time.Time
		_, _, _, _ = v0, v1, v2, v3
	}
}

func (x *TestType) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer1234
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		yym1 := z.EncBinary()
		_ = yym1
		if false {
		} else if z.HasExtensions() && z.EncExt(x) {
		} else {
			yysep2 := !z.EncBinary()
			yy2arr2 := z.EncBasicHandle().StructToArray
			var yyq2 [3]bool
			_, _, _ = yysep2, yyq2, yy2arr2
			const yyr2 bool = false
			yyq2[0] = x.Kind != ""
			yyq2[1] = x.APIVersion != ""
			yyq2[2] = true
			var yynn2 int
			if yyr2 || yy2arr2 {
				r.EncodeArrayStart(3)
			} else {
				yynn2 = 0
				for _, b := range yyq2 {
					if b {
						yynn2++
					}
				}
				r.EncodeMapStart(yynn2)
				yynn2 = 0
			}
			if yyr2 || yy2arr2 {
				z.EncSendContainerState(codecSelfer_containerArrayElem1234)
				if yyq2[0] {
					yym4 := z.EncBinary()
					_ = yym4
					if false {
					} else {
						r.EncodeString(codecSelferC_UTF81234, string(x.Kind))
					}
				} else {
					r.EncodeString(codecSelferC_UTF81234, "")
				}
			} else {
				if yyq2[0] {
					z.EncSendContainerState(codecSelfer_containerMapKey1234)
					r.EncodeString(codecSelferC_UTF81234, string("kind"))
					z.EncSendContainerState(codecSelfer_containerMapValue1234)
					yym5 := z.EncBinary()
					_ = yym5
					if false {
					} else {
						r.EncodeString(codecSelferC_UTF81234, string(x.Kind))
					}
				}
			}
			if yyr2 || yy2arr2 {
				z.EncSendContainerState(codecSelfer_containerArrayElem1234)
				if yyq2[1] {
					yym7 := z.EncBinary()
					_ = yym7
					if false {
					} else {
						r.EncodeString(codecSelferC_UTF81234, string(x.APIVersion))
					}
				} else {
					r.EncodeString(codecSelferC_UTF81234, "")
				}
			} else {
				if yyq2[1] {
					z.EncSendContainerState(codecSelfer_containerMapKey1234)
					r.EncodeString(codecSelferC_UTF81234, string("apiVersion"))
					z.EncSendContainerState(codecSelfer_containerMapValue1234)
					yym8 := z.EncBinary()
					_ = yym8
					if false {
					} else {
						r.EncodeString(codecSelferC_UTF81234, string(x.APIVersion))
					}
				}
			}
			if yyr2 || yy2arr2 {
				z.EncSendContainerState(codecSelfer_containerArrayElem1234)
				if yyq2[2] {
					yy10 := &x.ObjectMeta
					yym11 := z.EncBinary()
					_ = yym11
					if false {
					} else if z.HasExtensions() && z.EncExt(yy10) {
					} else {
						z.EncFallback(yy10)
					}
				} else {
					r.EncodeNil()
				}
			} else {
				if yyq2[2] {
					z.EncSendContainerState(codecSelfer_containerMapKey1234)
					r.EncodeString(codecSelferC_UTF81234, string("metadata"))
					z.EncSendContainerState(codecSelfer_containerMapValue1234)
					yy12 := &x.ObjectMeta
					yym13 := z.EncBinary()
					_ = yym13
					if false {
					} else if z.HasExtensions() && z.EncExt(yy12) {
					} else {
						z.EncFallback(yy12)
					}
				}
			}
			if yyr2 || yy2arr2 {
				z.EncSendContainerState(codecSelfer_containerArrayEnd1234)
			} else {
				z.EncSendContainerState(codecSelfer_containerMapEnd1234)
			}
		}
	}
}

func (x *TestType) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer1234
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	yym14 := z.DecBinary()
	_ = yym14
	if false {
	} else if z.HasExtensions() && z.DecExt(x) {
	} else {
		yyct15 := r.ContainerType()
		if yyct15 == codecSelferValueTypeMap1234 {
			yyl15 := r.ReadMapStart()
			if yyl15 == 0 {
				z.DecSendContainerState(codecSelfer_containerMapEnd1234)
			} else {
				x.codecDecodeSelfFromMap(yyl15, d)
			}
		} else if yyct15 == codecSelferValueTypeArray1234 {
			yyl15 := r.ReadArrayStart()
			if yyl15 == 0 {
				z.DecSendContainerState(codecSelfer_containerArrayEnd1234)
			} else {
				x.codecDecodeSelfFromArray(yyl15, d)
			}
		} else {
			panic(codecSelferOnlyMapOrArrayEncodeToStructErr1234)
		}
	}
}

func (x *TestType) codecDecodeSelfFromMap(l int, d *codec1978.Decoder) {
	var h codecSelfer1234
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yys16Slc = z.DecScratchBuffer() // default slice to decode into
	_ = yys16Slc
	var yyhl16 bool = l >= 0
	for yyj16 := 0; ; yyj16++ {
		if yyhl16 {
			if yyj16 >= l {
				break
			}
		} else {
			if r.CheckBreak() {
				break
			}
		}
		z.DecSendContainerState(codecSelfer_containerMapKey1234)
		yys16Slc = r.DecodeBytes(yys16Slc, true, true)
		yys16 := string(yys16Slc)
		z.DecSendContainerState(codecSelfer_containerMapValue1234)
		switch yys16 {
		case "kind":
			if r.TryDecodeAsNil() {
				x.Kind = ""
			} else {
				x.Kind = string(r.DecodeString())
			}
		case "apiVersion":
			if r.TryDecodeAsNil() {
				x.APIVersion = ""
			} else {
				x.APIVersion = string(r.DecodeString())
			}
		case "metadata":
			if r.TryDecodeAsNil() {
				x.ObjectMeta = pkg2_api.ObjectMeta{}
			} else {
				yyv19 := &x.ObjectMeta
				yym20 := z.DecBinary()
				_ = yym20
				if false {
				} else if z.HasExtensions() && z.DecExt(yyv19) {
				} else {
					z.DecFallback(yyv19, false)
				}
			}
		default:
			z.DecStructFieldNotFound(-1, yys16)
		} // end switch yys16
	} // end for yyj16
	z.DecSendContainerState(codecSelfer_containerMapEnd1234)
}

func (x *TestType) codecDecodeSelfFromArray(l int, d *codec1978.Decoder) {
	var h codecSelfer1234
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyj21 int
	var yyb21 bool
	var yyhl21 bool = l >= 0
	yyj21++
	if yyhl21 {
		yyb21 = yyj21 > l
	} else {
		yyb21 = r.CheckBreak()
	}
	if yyb21 {
		z.DecSendContainerState(codecSelfer_containerArrayEnd1234)
		return
	}
	z.DecSendContainerState(codecSelfer_containerArrayElem1234)
	if r.TryDecodeAsNil() {
		x.Kind = ""
	} else {
		x.Kind = string(r.DecodeString())
	}
	yyj21++
	if yyhl21 {
		yyb21 = yyj21 > l
	} else {
		yyb21 = r.CheckBreak()
	}
	if yyb21 {
		z.DecSendContainerState(codecSelfer_containerArrayEnd1234)
		return
	}
	z.DecSendContainerState(codecSelfer_containerArrayElem1234)
	if r.TryDecodeAsNil() {
		x.APIVersion = ""
	} else {
		x.APIVersion = string(r.DecodeString())
	}
	yyj21++
	if yyhl21 {
		yyb21 = yyj21 > l
	} else {
		yyb21 = r.CheckBreak()
	}
	if yyb21 {
		z.DecSendContainerState(codecSelfer_containerArrayEnd1234)
		return
	}
	z.DecSendContainerState(codecSelfer_containerArrayElem1234)
	if r.TryDecodeAsNil() {
		x.ObjectMeta = pkg2_api.ObjectMeta{}
	} else {
		yyv24 := &x.ObjectMeta
		yym25 := z.DecBinary()
		_ = yym25
		if false {
		} else if z.HasExtensions() && z.DecExt(yyv24) {
		} else {
			z.DecFallback(yyv24, false)
		}
	}
	for {
		yyj21++
		if yyhl21 {
			yyb21 = yyj21 > l
		} else {
			yyb21 = r.CheckBreak()
		}
		if yyb21 {
			break
		}
		z.DecSendContainerState(codecSelfer_containerArrayElem1234)
		z.DecStructFieldNotFound(yyj21-1, "")
	}
	z.DecSendContainerState(codecSelfer_containerArrayEnd1234)
}

func (x *TestTypeList) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer1234
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		yym26 := z.EncBinary()
		_ = yym26
		if false {
		} else if z.HasExtensions() && z.EncExt(x) {
		} else {
			yysep27 := !z.EncBinary()
			yy2arr27 := z.EncBasicHandle().StructToArray
			var yyq27 [4]bool
			_, _, _ = yysep27, yyq27, yy2arr27
			const yyr27 bool = false
			yyq27[0] = x.Kind != ""
			yyq27[1] = x.APIVersion != ""
			yyq27[2] = true
			var yynn27 int
			if yyr27 || yy2arr27 {
				r.EncodeArrayStart(4)
			} else {
				yynn27 = 1
				for _, b := range yyq27 {
					if b {
						yynn27++
					}
				}
				r.EncodeMapStart(yynn27)
				yynn27 = 0
			}
			if yyr27 || yy2arr27 {
				z.EncSendContainerState(codecSelfer_containerArrayElem1234)
				if yyq27[0] {
					yym29 := z.EncBinary()
					_ = yym29
					if false {
					} else {
						r.EncodeString(codecSelferC_UTF81234, string(x.Kind))
					}
				} else {
					r.EncodeString(codecSelferC_UTF81234, "")
				}
			} else {
				if yyq27[0] {
					z.EncSendContainerState(codecSelfer_containerMapKey1234)
					r.EncodeString(codecSelferC_UTF81234, string("kind"))
					z.EncSendContainerState(codecSelfer_containerMapValue1234)
					yym30 := z.EncBinary()
					_ = yym30
					if false {
					} else {
						r.EncodeString(codecSelferC_UTF81234, string(x.Kind))
					}
				}
			}
			if yyr27 || yy2arr27 {
				z.EncSendContainerState(codecSelfer_containerArrayElem1234)
				if yyq27[1] {
					yym32 := z.EncBinary()
					_ = yym32
					if false {
					} else {
						r.EncodeString(codecSelferC_UTF81234, string(x.APIVersion))
					}
				} else {
					r.EncodeString(codecSelferC_UTF81234, "")
				}
			} else {
				if yyq27[1] {
					z.EncSendContainerState(codecSelfer_containerMapKey1234)
					r.EncodeString(codecSelferC_UTF81234, string("apiVersion"))
					z.EncSendContainerState(codecSelfer_containerMapValue1234)
					yym33 := z.EncBinary()
					_ = yym33
					if false {
					} else {
						r.EncodeString(codecSelferC_UTF81234, string(x.APIVersion))
					}
				}
			}
			if yyr27 || yy2arr27 {
				z.EncSendContainerState(codecSelfer_containerArrayElem1234)
				if yyq27[2] {
					yy35 := &x.ListMeta
					yym36 := z.EncBinary()
					_ = yym36
					if false {
					} else if z.HasExtensions() && z.EncExt(yy35) {
					} else {
						z.EncFallback(yy35)
					}
				} else {
					r.EncodeNil()
				}
			} else {
				if yyq27[2] {
					z.EncSendContainerState(codecSelfer_containerMapKey1234)
					r.EncodeString(codecSelferC_UTF81234, string("metadata"))
					z.EncSendContainerState(codecSelfer_containerMapValue1234)
					yy37 := &x.ListMeta
					yym38 := z.EncBinary()
					_ = yym38
					if false {
					} else if z.HasExtensions() && z.EncExt(yy37) {
					} else {
						z.EncFallback(yy37)
					}
				}
			}
			if yyr27 || yy2arr27 {
				z.EncSendContainerState(codecSelfer_containerArrayElem1234)
				if x.Items == nil {
					r.EncodeNil()
				} else {
					yym40 := z.EncBinary()
					_ = yym40
					if false {
					} else {
						h.encSliceTestType(([]TestType)(x.Items), e)
					}
				}
			} else {
				z.EncSendContainerState(codecSelfer_containerMapKey1234)
				r.EncodeString(codecSelferC_UTF81234, string("items"))
				z.EncSendContainerState(codecSelfer_containerMapValue1234)
				if x.Items == nil {
					r.EncodeNil()
				} else {
					yym41 := z.EncBinary()
					_ = yym41
					if false {
					} else {
						h.encSliceTestType(([]TestType)(x.Items), e)
					}
				}
			}
			if yyr27 || yy2arr27 {
				z.EncSendContainerState(codecSelfer_containerArrayEnd1234)
			} else {
				z.EncSendContainerState(codecSelfer_containerMapEnd1234)
			}
		}
	}
}

func (x *TestTypeList) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer1234
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	yym42 := z.DecBinary()
	_ = yym42
	if false {
	} else if z.HasExtensions() && z.DecExt(x) {
	} else {
		yyct43 := r.ContainerType()
		if yyct43 == codecSelferValueTypeMap1234 {
			yyl43 := r.ReadMapStart()
			if yyl43 == 0 {
				z.DecSendContainerState(codecSelfer_containerMapEnd1234)
			} else {
				x.codecDecodeSelfFromMap(yyl43, d)
			}
		} else if yyct43 == codecSelferValueTypeArray1234 {
			yyl43 := r.ReadArrayStart()
			if yyl43 == 0 {
				z.DecSendContainerState(codecSelfer_containerArrayEnd1234)
			} else {
				x.codecDecodeSelfFromArray(yyl43, d)
			}
		} else {
			panic(codecSelferOnlyMapOrArrayEncodeToStructErr1234)
		}
	}
}

func (x *TestTypeList) codecDecodeSelfFromMap(l int, d *codec1978.Decoder) {
	var h codecSelfer1234
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yys44Slc = z.DecScratchBuffer() // default slice to decode into
	_ = yys44Slc
	var yyhl44 bool = l >= 0
	for yyj44 := 0; ; yyj44++ {
		if yyhl44 {
			if yyj44 >= l {
				break
			}
		} else {
			if r.CheckBreak() {
				break
			}
		}
		z.DecSendContainerState(codecSelfer_containerMapKey1234)
		yys44Slc = r.DecodeBytes(yys44Slc, true, true)
		yys44 := string(yys44Slc)
		z.DecSendContainerState(codecSelfer_containerMapValue1234)
		switch yys44 {
		case "kind":
			if r.TryDecodeAsNil() {
				x.Kind = ""
			} else {
				x.Kind = string(r.DecodeString())
			}
		case "apiVersion":
			if r.TryDecodeAsNil() {
				x.APIVersion = ""
			} else {
				x.APIVersion = string(r.DecodeString())
			}
		case "metadata":
			if r.TryDecodeAsNil() {
				x.ListMeta = pkg1_unversioned.ListMeta{}
			} else {
				yyv47 := &x.ListMeta
				yym48 := z.DecBinary()
				_ = yym48
				if false {
				} else if z.HasExtensions() && z.DecExt(yyv47) {
				} else {
					z.DecFallback(yyv47, false)
				}
			}
		case "items":
			if r.TryDecodeAsNil() {
				x.Items = nil
			} else {
				yyv49 := &x.Items
				yym50 := z.DecBinary()
				_ = yym50
				if false {
				} else {
					h.decSliceTestType((*[]TestType)(yyv49), d)
				}
			}
		default:
			z.DecStructFieldNotFound(-1, yys44)
		} // end switch yys44
	} // end for yyj44
	z.DecSendContainerState(codecSelfer_containerMapEnd1234)
}

func (x *TestTypeList) codecDecodeSelfFromArray(l int, d *codec1978.Decoder) {
	var h codecSelfer1234
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyj51 int
	var yyb51 bool
	var yyhl51 bool = l >= 0
	yyj51++
	if yyhl51 {
		yyb51 = yyj51 > l
	} else {
		yyb51 = r.CheckBreak()
	}
	if yyb51 {
		z.DecSendContainerState(codecSelfer_containerArrayEnd1234)
		return
	}
	z.DecSendContainerState(codecSelfer_containerArrayElem1234)
	if r.TryDecodeAsNil() {
		x.Kind = ""
	} else {
		x.Kind = string(r.DecodeString())
	}
	yyj51++
	if yyhl51 {
		yyb51 = yyj51 > l
	} else {
		yyb51 = r.CheckBreak()
	}
	if yyb51 {
		z.DecSendContainerState(codecSelfer_containerArrayEnd1234)
		return
	}
	z.DecSendContainerState(codecSelfer_containerArrayElem1234)
	if r.TryDecodeAsNil() {
		x.APIVersion = ""
	} else {
		x.APIVersion = string(r.DecodeString())
	}
	yyj51++
	if yyhl51 {
		yyb51 = yyj51 > l
	} else {
		yyb51 = r.CheckBreak()
	}
	if yyb51 {
		z.DecSendContainerState(codecSelfer_containerArrayEnd1234)
		return
	}
	z.DecSendContainerState(codecSelfer_containerArrayElem1234)
	if r.TryDecodeAsNil() {
		x.ListMeta = pkg1_unversioned.ListMeta{}
	} else {
		yyv54 := &x.ListMeta
		yym55 := z.DecBinary()
		_ = yym55
		if false {
		} else if z.HasExtensions() && z.DecExt(yyv54) {
		} else {
			z.DecFallback(yyv54, false)
		}
	}
	yyj51++
	if yyhl51 {
		yyb51 = yyj51 > l
	} else {
		yyb51 = r.CheckBreak()
	}
	if yyb51 {
		z.DecSendContainerState(codecSelfer_containerArrayEnd1234)
		return
	}
	z.DecSendContainerState(codecSelfer_containerArrayElem1234)
	if r.TryDecodeAsNil() {
		x.Items = nil
	} else {
		yyv56 := &x.Items
		yym57 := z.DecBinary()
		_ = yym57
		if false {
		} else {
			h.decSliceTestType((*[]TestType)(yyv56), d)
		}
	}
	for {
		yyj51++
		if yyhl51 {
			yyb51 = yyj51 > l
		} else {
			yyb51 = r.CheckBreak()
		}
		if yyb51 {
			break
		}
		z.DecSendContainerState(codecSelfer_containerArrayElem1234)
		z.DecStructFieldNotFound(yyj51-1, "")
	}
	z.DecSendContainerState(codecSelfer_containerArrayEnd1234)
}

func (x codecSelfer1234) encSliceTestType(v []TestType, e *codec1978.Encoder) {
	var h codecSelfer1234
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	r.EncodeArrayStart(len(v))
	for _, yyv58 := range v {
		z.EncSendContainerState(codecSelfer_containerArrayElem1234)
		yy59 := &yyv58
		yy59.CodecEncodeSelf(e)
	}
	z.EncSendContainerState(codecSelfer_containerArrayEnd1234)
}

func (x codecSelfer1234) decSliceTestType(v *[]TestType, d *codec1978.Decoder) {
	var h codecSelfer1234
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r

	yyv60 := *v
	yyh60, yyl60 := z.DecSliceHelperStart()
	var yyc60 bool
	if yyl60 == 0 {
		if yyv60 == nil {
			yyv60 = []TestType{}
			yyc60 = true
		} else if len(yyv60) != 0 {
			yyv60 = yyv60[:0]
			yyc60 = true
		}
	} else if yyl60 > 0 {
		var yyrr60, yyrl60 int
		var yyrt60 bool
		if yyl60 > cap(yyv60) {

			yyrg60 := len(yyv60) > 0
			yyv260 := yyv60
			yyrl60, yyrt60 = z.DecInferLen(yyl60, z.DecBasicHandle().MaxInitLen, 192)
			if yyrt60 {
				if yyrl60 <= cap(yyv60) {
					yyv60 = yyv60[:yyrl60]
				} else {
					yyv60 = make([]TestType, yyrl60)
				}
			} else {
				yyv60 = make([]TestType, yyrl60)
			}
			yyc60 = true
			yyrr60 = len(yyv60)
			if yyrg60 {
				copy(yyv60, yyv260)
			}
		} else if yyl60 != len(yyv60) {
			yyv60 = yyv60[:yyl60]
			yyc60 = true
		}
		yyj60 := 0
		for ; yyj60 < yyrr60; yyj60++ {
			yyh60.ElemContainerState(yyj60)
			if r.TryDecodeAsNil() {
				yyv60[yyj60] = TestType{}
			} else {
				yyv61 := &yyv60[yyj60]
				yyv61.CodecDecodeSelf(d)
			}

		}
		if yyrt60 {
			for ; yyj60 < yyl60; yyj60++ {
				yyv60 = append(yyv60, TestType{})
				yyh60.ElemContainerState(yyj60)
				if r.TryDecodeAsNil() {
					yyv60[yyj60] = TestType{}
				} else {
					yyv62 := &yyv60[yyj60]
					yyv62.CodecDecodeSelf(d)
				}

			}
		}

	} else {
		yyj60 := 0
		for ; !r.CheckBreak(); yyj60++ {

			if yyj60 >= len(yyv60) {
				yyv60 = append(yyv60, TestType{}) // var yyz60 TestType
				yyc60 = true
			}
			yyh60.ElemContainerState(yyj60)
			if yyj60 < len(yyv60) {
				if r.TryDecodeAsNil() {
					yyv60[yyj60] = TestType{}
				} else {
					yyv63 := &yyv60[yyj60]
					yyv63.CodecDecodeSelf(d)
				}

			} else {
				z.DecSwallow()
			}

		}
		if yyj60 < len(yyv60) {
			yyv60 = yyv60[:yyj60]
			yyc60 = true
		} else if yyj60 == 0 && yyv60 == nil {
			yyv60 = []TestType{}
			yyc60 = true
		}
	}
	yyh60.End()
	if yyc60 {
		*v = yyv60
	}
}
