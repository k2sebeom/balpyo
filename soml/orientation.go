package soml

import "encoding/binary"

// 이 파일은 **JPEG의 EXIF 방향**이다 — 휴대폰 사진은 픽셀을 센서 방향 그대로 저장하고 "이렇게
// 돌려서 보라"를 EXIF 태그(0x0112) 하나로 적는다.
//
// 브라우저는 그 태그를 따른다 (`image-orientation: from-image`가 기본값이다): 웹은 돌린 그림을
// 보여주고, 지오메트리 패스도 돌린 박스를 잰다. PowerPoint는 그림의 EXIF를 보지 않으므로 PPTX가
// 같은 그림을 보여주려면 돌림을 `xfrm`의 회전·대칭으로 옮겨야 한다. 바이트는 고치지 않는다 —
// 다시 인코딩하면 저자가 커밋한 파일과 산출물의 그림이 다른 것이 된다.

// Orientation은 EXIF 방향 태그 값이다 (1–8). 0은 태그가 없다는 뜻이고 1과 같이 그린다.
//
// 값마다 "저장된 픽셀을 어떻게 놓으면 보이는 그림이 되는가"가 정해져 있다 — 좌우 대칭을 먼저
// 하고 시계 방향으로 돌린다 (Rotation · FlipH).
//
//	1 그대로          2 좌우 대칭        3 180°           4 상하 대칭
//	5 좌우 대칭+270°   6 90°             7 좌우 대칭+90°   8 270°
type Orientation int

// orientationTransforms는 값 → (시계 방향 회전 각도, 좌우 대칭)이다. 상하 대칭(4)은 좌우 대칭 +
// 180°와 같은 그림이므로 대칭은 한 축만 쓴다 — 변환이 두 성분뿐이어야 받는 쪽이 합성할 경우가 준다.
var orientationTransforms = [9]struct {
	rotate int
	flipH  bool
}{
	0: {0, false},
	1: {0, false},
	2: {0, true},
	3: {180, false},
	4: {180, true},
	5: {270, true},
	6: {90, false},
	7: {90, true},
	8: {270, false},
}

func (o Orientation) valid() bool {
	return o >= 0 && int(o) < len(orientationTransforms)
}

// Rotation은 대칭 **다음에** 거는 시계 방향 회전 각도다 (0 · 90 · 180 · 270).
func (o Orientation) Rotation() int {
	if !o.valid() {
		return 0
	}
	return orientationTransforms[o].rotate
}

// FlipH는 회전 **전에** 좌우 대칭을 하는가다.
func (o Orientation) FlipH() bool {
	if !o.valid() {
		return false
	}
	return orientationTransforms[o].flipH
}

// SwapsAxes는 보이는 그림의 가로·세로가 저장된 픽셀과 뒤바뀌는가다 (5–8 — 90°·270° 회전).
func (o Orientation) SwapsAxes() bool {
	return o.Rotation()%180 != 0
}

// exifOrientationTag는 IFD0의 방향 태그 번호다.
const exifOrientationTag = 0x0112

// JPEGOrientation은 JPEG 바이트의 EXIF 방향을 읽는다 — 태그가 없거나 읽을 수 없으면 0이다.
//
// 헤더만 훑는다: 마커를 따라가다 APP1(`Exif\0\0`)을 만나면 TIFF 헤더 → IFD0에서 태그 하나를
// 찾고, 이미지 데이터(SOS)에 닿으면 멈춘다. 깨진 EXIF는 진단이 아니라 0이다 — 브라우저도 읽지
// 못한 태그는 무시하고 저장된 그대로 그린다.
func JPEGOrientation(data []byte) Orientation {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 0
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return 0
		}
		marker := data[i+1]
		switch {
		case marker == 0xFF:
			// 채움 바이트 — 마커 앞에 몇 개든 올 수 있다.
			i++
			continue
		case marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7):
			// 길이 없는 단독 마커.
			i += 2
			continue
		case marker == 0xDA || marker == 0xD9:
			// 이미지 데이터의 시작(SOS) · 끝(EOI) — 그 뒤에는 메타데이터가 없다.
			return 0
		}
		size := int(binary.BigEndian.Uint16(data[i+2:]))
		if size < 2 || i+2+size > len(data) {
			return 0
		}
		segment := data[i+4 : i+2+size]
		if marker == 0xE1 && len(segment) >= 6 && string(segment[:6]) == "Exif\x00\x00" {
			if o := tiffOrientation(segment[6:]); o != 0 {
				return o
			}
		}
		i += 2 + size
	}
	return 0
}

// tiffOrientation은 EXIF의 TIFF 블록에서 IFD0의 방향 태그를 찾는다.
func tiffOrientation(tiff []byte) Orientation {
	if len(tiff) < 8 {
		return 0
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0
	}
	if order.Uint16(tiff[2:]) != 42 {
		return 0
	}
	ifd := int(order.Uint32(tiff[4:]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 0
	}
	count := int(order.Uint16(tiff[ifd:]))
	for n := 0; n < count; n++ {
		entry := ifd + 2 + n*12
		if entry+12 > len(tiff) {
			return 0
		}
		if order.Uint16(tiff[entry:]) != exifOrientationTag {
			continue
		}
		// 형식은 SHORT(3) 하나다 — 값은 값 칸의 앞 2바이트에 바로 들어 있다.
		if order.Uint16(tiff[entry+2:]) != 3 || order.Uint32(tiff[entry+4:]) != 1 {
			return 0
		}
		o := Orientation(order.Uint16(tiff[entry+8:]))
		if o < 1 || o > 8 {
			return 0
		}
		return o
	}
	return 0
}
