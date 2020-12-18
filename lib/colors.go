package data

import "image/color"

// An approximation of Atari palette colors
var rgbcolors = [3 * 256]int{
	4, 4, 4,
	15, 15, 15,
	27, 27, 27,
	39, 39, 39,
	51, 51, 51,
	65, 65, 65,
	79, 79, 79,
	94, 94, 94,
	104, 104, 104,
	120, 120, 120,
	137, 137, 137,
	154, 154, 154,
	171, 171, 171,
	191, 191, 191,
	211, 211, 211,
	234, 234, 234,
	44, 1, -48,
	56, 8, -37,
	67, 19, -26,
	79, 32, -15,
	91, 44, -3,
	105, 58, 1,
	119, 72, 11,
	133, 86, 25,
	144, 97, 36,
	159, 113, 52,
	176, 130, 70,
	194, 147, 87,
	211, 164, 104,
	230, 184, 124,
	250, 204, 144,
	273, 227, 167,
	56, -2, -16,
	67, 1, -5,
	79, 7, 0,
	91, 19, 5,
	103, 31, 16,
	116, 45, 30,
	131, 60, 45,
	145, 74, 59,
	155, 84, 70,
	171, 100, 86,
	188, 118, 103,
	205, 135, 120,
	222, 152, 138,
	241, 171, 157,
	261, 192, 177,
	284, 215, 200,
	57, -6, 0,
	69, 0, 2,
	80, 4, 10,
	92, 15, 23,
	104, 27, 35,
	118, 41, 49,
	132, 55, 63,
	146, 70, 78,
	157, 80, 88,
	172, 96, 104,
	189, 113, 121,
	206, 131, 138,
	224, 148, 156,
	243, 167, 175,
	263, 187, 195,
	286, 210, 218,
	53, -13, 36,
	64, -2, 48,
	76, 0, 59,
	88, 7, 71,
	100, 19, 83,
	113, 33, 97,
	128, 48, 111,
	142, 62, 126,
	152, 73, 136,
	168, 89, 152,
	185, 106, 169,
	202, 123, 186,
	219, 141, 203,
	239, 160, 223,
	259, 180, 243,
	281, 203, 265,
	42, -13, 68,
	53, -3, 80,
	65, 0, 91,
	77, 7, 103,
	89, 19, 115,
	102, 33, 129,
	117, 47, 143,
	131, 62, 157,
	141, 72, 167,
	157, 88, 183,
	174, 106, 200,
	191, 123, 217,
	208, 140, 234,
	228, 160, 253,
	248, 180, 273,
	271, 203, 296,
	24, -9, 90,
	36, 0, 101,
	47, 2, 112,
	60, 11, 124,
	72, 24, 136,
	85, 38, 150,
	100, 52, 164,
	114, 67, 178,
	124, 77, 188,
	140, 93, 204,
	157, 110, 221,
	174, 128, 238,
	192, 145, 255,
	211, 164, 274,
	231, 184, 294,
	254, 207, 317,
	-3, 1, 90,
	0, 9, 101,
	6, 20, 112,
	18, 33, 124,
	30, 45, 136,
	44, 59, 150,
	59, 73, 164,
	73, 87, 178,
	84, 98, 188,
	99, 114, 204,
	117, 131, 221,
	134, 148, 238,
	151, 165, 255,
	171, 185, 274,
	191, 205, 294,
	214, 228, 317,
	-19, 10, 68,
	-8, 22, 80,
	0, 34, 91,
	3, 46, 103,
	13, 58, 115,
	27, 72, 129,
	41, 86, 143,
	56, 100, 157,
	66, 111, 167,
	82, 127, 183,
	100, 144, 200,
	117, 161, 217,
	134, 178, 234,
	154, 197, 253,
	174, 217, 273,
	197, 240, 296,
	-30, 22, 36,
	-19, 34, 48,
	-8, 45, 59,
	0, 58, 71,
	3, 70, 83,
	15, 83, 97,
	30, 98, 111,
	45, 112, 126,
	55, 122, 136,
	71, 138, 152,
	89, 155, 169,
	106, 173, 186,
	123, 190, 203,
	143, 209, 223,
	163, 229, 243,
	186, 252, 265,
	-34, 34, 0,
	-23, 46, 2,
	-12, 57, 10,
	-1, 69, 23,
	1, 81, 35,
	11, 95, 49,
	26, 109, 63,
	40, 124, 78,
	51, 134, 88,
	67, 150, 104,
	84, 167, 121,
	102, 184, 138,
	119, 201, 156,
	138, 220, 175,
	159, 240, 195,
	182, 263, 218,
	-21, 38, -48,
	-11, 49, -37,
	0, 60, -26,
	2, 73, -15,
	10, 85, -3,
	25, 98, 1,
	39, 113, 11,
	54, 127, 25,
	64, 137, 36,
	80, 153, 52,
	97, 170, 70,
	115, 187, 87,
	132, 204, 104,
	151, 224, 124,
	172, 244, 144,
	195, 267, 167,
	-4, 33, -69,
	0, 44, -58,
	5, 56, -47,
	17, 68, -36,
	29, 80, -25,
	43, 93, -12,
	57, 108, 0,
	72, 122, 4,
	82, 132, 13,
	98, 148, 29,
	116, 165, 47,
	133, 182, 64,
	150, 200, 82,
	169, 219, 101,
	190, 239, 122,
	213, 262, 145,
	4, 23, -76,
	15, 34, -66,
	27, 46, -55,
	39, 58, -44,
	51, 70, -32,
	65, 84, -19,
	79, 98, -6,
	94, 113, 0,
	104, 123, 6,
	120, 139, 21,
	137, 156, 39,
	154, 173, 56,
	171, 190, 74,
	191, 210, 94,
	211, 230, 114,
	234, 252, 137,
	26, 10, -69,
	37, 22, -58,
	49, 33, -47,
	61, 45, -36,
	73, 58, -25,
	87, 71, -12,
	101, 86, 0,
	115, 100, 4,
	126, 110, 13,
	141, 126, 29,
	159, 143, 47,
	176, 161, 64,
	193, 178, 82,
	212, 197, 101,
	232, 217, 122,
	255, 240, 145,
	44, 1, -48,
	56, 8, -37,
	67, 19, -26,
	79, 32, -15,
	91, 44, -3,
	105, 58, 1,
	119, 72, 11,
	133, 86, 25,
	144, 97, 36,
	159, 113, 52,
	176, 130, 70,
	194, 147, 87,
	211, 164, 104,
	230, 184, 124,
	250, 204, 144,
	273, 227, 167,
}
var RGBPalette [256]color.RGBA

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func init() {
	for i, val := range rgbcolors {
		switch i % 3 {
		case 0:
			RGBPalette[i/3].R = uint8(clamp(val, 0, 255))
		case 1:
			RGBPalette[i/3].G = uint8(clamp(val, 0, 255))
		case 2:
			RGBPalette[i/3].B = uint8(clamp(val, 0, 255))
			RGBPalette[i/3].A = 255
		}
	}
}