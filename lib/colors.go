package lib

import "image/color"

// An approximation of Atari palette colors + a transparent color as 256.
var RGBPalette = [257]color.RGBA{
	{4, 4, 4, 255},
	{15, 15, 15, 255},
	{27, 27, 27, 255},
	{39, 39, 39, 255},
	{51, 51, 51, 255},
	{65, 65, 65, 255},
	{79, 79, 79, 255},
	{94, 94, 94, 255},
	{104, 104, 104, 255},
	{120, 120, 120, 255},
	{137, 137, 137, 255},
	{154, 154, 154, 255},
	{171, 171, 171, 255},
	{191, 191, 191, 255},
	{211, 211, 211, 255},
	{234, 234, 234, 255},
	{44, 1, 0, 255},
	{56, 8, 0, 255},
	{67, 19, 0, 255},
	{79, 32, 0, 255},
	{91, 44, 0, 255},
	{105, 58, 1, 255},
	{119, 72, 11, 255},
	{133, 86, 25, 255},
	{144, 97, 36, 255},
	{159, 113, 52, 255},
	{176, 130, 70, 255},
	{194, 147, 87, 255},
	{211, 164, 104, 255},
	{230, 184, 124, 255},
	{250, 204, 144, 255},
	{255, 227, 167, 255},
	{56, 0, 0, 255},
	{67, 1, 0, 255},
	{79, 7, 0, 255},
	{91, 19, 5, 255},
	{103, 31, 16, 255},
	{116, 45, 30, 255},
	{131, 60, 45, 255},
	{145, 74, 59, 255},
	{155, 84, 70, 255},
	{171, 100, 86, 255},
	{188, 118, 103, 255},
	{205, 135, 120, 255},
	{222, 152, 138, 255},
	{241, 171, 157, 255},
	{255, 192, 177, 255},
	{255, 215, 200, 255},
	{57, 0, 0, 255},
	{69, 0, 2, 255},
	{80, 4, 10, 255},
	{92, 15, 23, 255},
	{104, 27, 35, 255},
	{118, 41, 49, 255},
	{132, 55, 63, 255},
	{146, 70, 78, 255},
	{157, 80, 88, 255},
	{172, 96, 104, 255},
	{189, 113, 121, 255},
	{206, 131, 138, 255},
	{224, 148, 156, 255},
	{243, 167, 175, 255},
	{255, 187, 195, 255},
	{255, 210, 218, 255},
	{53, 0, 36, 255},
	{64, 0, 48, 255},
	{76, 0, 59, 255},
	{88, 7, 71, 255},
	{100, 19, 83, 255},
	{113, 33, 97, 255},
	{128, 48, 111, 255},
	{142, 62, 126, 255},
	{152, 73, 136, 255},
	{168, 89, 152, 255},
	{185, 106, 169, 255},
	{202, 123, 186, 255},
	{219, 141, 203, 255},
	{239, 160, 223, 255},
	{255, 180, 243, 255},
	{255, 203, 255, 255},
	{42, 0, 68, 255},
	{53, 0, 80, 255},
	{65, 0, 91, 255},
	{77, 7, 103, 255},
	{89, 19, 115, 255},
	{102, 33, 129, 255},
	{117, 47, 143, 255},
	{131, 62, 157, 255},
	{141, 72, 167, 255},
	{157, 88, 183, 255},
	{174, 106, 200, 255},
	{191, 123, 217, 255},
	{208, 140, 234, 255},
	{228, 160, 253, 255},
	{248, 180, 255, 255},
	{255, 203, 255, 255},
	{24, 0, 90, 255},
	{36, 0, 101, 255},
	{47, 2, 112, 255},
	{60, 11, 124, 255},
	{72, 24, 136, 255},
	{85, 38, 150, 255},
	{100, 52, 164, 255},
	{114, 67, 178, 255},
	{124, 77, 188, 255},
	{140, 93, 204, 255},
	{157, 110, 221, 255},
	{174, 128, 238, 255},
	{192, 145, 255, 255},
	{211, 164, 255, 255},
	{231, 184, 255, 255},
	{254, 207, 255, 255},
	{0, 1, 90, 255},
	{0, 9, 101, 255},
	{6, 20, 112, 255},
	{18, 33, 124, 255},
	{30, 45, 136, 255},
	{44, 59, 150, 255},
	{59, 73, 164, 255},
	{73, 87, 178, 255},
	{84, 98, 188, 255},
	{99, 114, 204, 255},
	{117, 131, 221, 255},
	{134, 148, 238, 255},
	{151, 165, 255, 255},
	{171, 185, 255, 255},
	{191, 205, 255, 255},
	{255, 228, 255, 255},
	{0, 10, 68, 255},
	{0, 22, 80, 255},
	{0, 34, 91, 255},
	{3, 46, 103, 255},
	{13, 58, 115, 255},
	{27, 72, 129, 255},
	{41, 86, 143, 255},
	{56, 100, 157, 255},
	{66, 111, 167, 255},
	{82, 127, 183, 255},
	{100, 144, 200, 255},
	{117, 161, 217, 255},
	{134, 178, 234, 255},
	{154, 197, 253, 255},
	{174, 217, 255, 255},
	{197, 240, 255, 255},
	{0, 22, 36, 255},
	{0, 34, 48, 255},
	{0, 45, 59, 255},
	{0, 58, 71, 255},
	{3, 70, 83, 255},
	{15, 83, 97, 255},
	{30, 98, 111, 255},
	{45, 112, 126, 255},
	{55, 122, 136, 255},
	{71, 138, 152, 255},
	{89, 155, 169, 255},
	{106, 173, 186, 255},
	{123, 190, 203, 255},
	{143, 209, 223, 255},
	{163, 229, 243, 255},
	{186, 252, 255, 255},
	{0, 34, 0, 255},
	{0, 46, 2, 255},
	{0, 57, 10, 255},
	{0, 69, 23, 255},
	{1, 81, 35, 255},
	{11, 95, 49, 255},
	{26, 109, 63, 255},
	{40, 124, 78, 255},
	{51, 134, 88, 255},
	{67, 150, 104, 255},
	{84, 167, 121, 255},
	{102, 184, 138, 255},
	{119, 201, 156, 255},
	{138, 220, 175, 255},
	{159, 240, 195, 255},
	{182, 255, 218, 255},
	{0, 38, 0, 255},
	{0, 49, 0, 255},
	{0, 60, 0, 255},
	{2, 73, 0, 255},
	{10, 85, 0, 255},
	{25, 98, 1, 255},
	{39, 113, 11, 255},
	{54, 127, 25, 255},
	{64, 137, 36, 255},
	{80, 153, 52, 255},
	{97, 170, 70, 255},
	{115, 187, 87, 255},
	{132, 204, 104, 255},
	{151, 224, 124, 255},
	{172, 244, 144, 255},
	{195, 255, 167, 255},
	{0, 33, 0, 255},
	{0, 44, 0, 255},
	{5, 56, 0, 255},
	{17, 68, 0, 255},
	{29, 80, 0, 255},
	{43, 93, 0, 255},
	{57, 108, 0, 255},
	{72, 122, 4, 255},
	{82, 132, 13, 255},
	{98, 148, 29, 255},
	{116, 165, 47, 255},
	{133, 182, 64, 255},
	{150, 200, 82, 255},
	{169, 219, 101, 255},
	{190, 239, 122, 255},
	{213, 255, 145, 255},
	{4, 23, 0, 255},
	{15, 34, 0, 255},
	{27, 46, 0, 255},
	{39, 58, 0, 255},
	{51, 70, 0, 255},
	{65, 84, 0, 255},
	{79, 98, 0, 255},
	{94, 113, 0, 255},
	{104, 123, 6, 255},
	{120, 139, 21, 255},
	{137, 156, 39, 255},
	{154, 173, 56, 255},
	{171, 190, 74, 255},
	{191, 210, 94, 255},
	{211, 230, 114, 255},
	{234, 252, 137, 255},
	{26, 10, 0, 255},
	{37, 22, 0, 255},
	{49, 33, 0, 255},
	{61, 45, 0, 255},
	{73, 58, 0, 255},
	{87, 71, 0, 255},
	{101, 86, 0, 255},
	{115, 100, 4, 255},
	{126, 110, 13, 255},
	{141, 126, 29, 255},
	{159, 143, 47, 255},
	{176, 161, 64, 255},
	{193, 178, 82, 255},
	{212, 197, 101, 255},
	{232, 217, 122, 255},
	{255, 240, 145, 255},
	{44, 1, 0, 255},
	{56, 8, 0, 255},
	{67, 19, 0, 255},
	{79, 32, 0, 255},
	{91, 44, 0, 255},
	{105, 58, 1, 255},
	{119, 72, 11, 255},
	{133, 86, 25, 255},
	{144, 97, 36, 255},
	{159, 113, 52, 255},
	{176, 130, 70, 255},
	{194, 147, 87, 255},
	{211, 164, 104, 255},
	{230, 184, 124, 255},
	{250, 204, 144, 255},
	{255, 227, 167, 255},
	{0, 0, 0, 0},
}
