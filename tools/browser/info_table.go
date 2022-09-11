package main

import (
	"strconv"

	"github.com/pwiecz/command_series/lib"
	"github.com/pwiecz/go-fltk"
)

type InfoTable struct {
	*fltk.Pack
	perUnitTable      *fltk.TableRow
	perTerrainTable   *fltk.TableRow
	perFormationTable *fltk.TableRow
	perGeneralTable   *fltk.TableRow
	gameData          *lib.GameData
	scenarioData      *lib.ScenarioData
	selectedScenario  int
	side0GeneralCount int
	side1GeneralCount int
}

func NewInfoTable(x, y, w, h int) *InfoTable {
	t := &InfoTable{}
	t.Pack = fltk.NewPack(x, y, w, h)
	t.Pack.SetType(fltk.VERTICAL)
	t.Pack.SetSpacing(5)
	t.perUnitTable = fltk.NewTableRow(0, 0, w, 400)
	t.perUnitTable.SetColumnCount(17)
	t.perUnitTable.SetRowCount(len(perUnitFieldNames))
	t.perUnitTable.SetDrawCellCallback(t.perUnitDrawCallback)
	t.perUnitTable.EnableColumnHeaders()
	t.perUnitTable.AllowColumnResizing()
	t.Pack.Add(t.perUnitTable)
	t.perTerrainTable = fltk.NewTableRow(0, 0, w, 136)
	t.perTerrainTable.SetColumnCount(9)
	t.perTerrainTable.SetRowCount(len(perTerrainFieldNames))
	t.perTerrainTable.SetDrawCellCallback(t.perTerrainDrawCallback)
	t.perTerrainTable.EnableColumnHeaders()
	t.Pack.Add(t.perTerrainTable)
	t.perFormationTable = fltk.NewTableRow(0, 0, w, 180)
	t.perFormationTable.SetColumnCount(9)
	t.perFormationTable.SetRowCount(len(perFormationFieldNames))
	t.perFormationTable.SetDrawCellCallback(t.perFormationDrawCallback)
	t.perFormationTable.EnableColumnHeaders()
	t.Pack.Add(t.perFormationTable)
	t.perGeneralTable = fltk.NewTableRow(0, 0, w, 200)
	t.perGeneralTable.SetRowCount(len(perGeneralFieldNames))
	t.perGeneralTable.SetDrawCellCallback(t.perGeneralDrawCallback)
	t.perGeneralTable.EnableColumnHeaders()
	t.perGeneralTable.AllowColumnResizing()
	t.Pack.Add(t.perGeneralTable)
	t.Pack.End()
	t.Pack.Resizable(t.perUnitTable)
	return t
}

func (t *InfoTable) SetGameData(gameData *lib.GameData, scenarioData *lib.ScenarioData, selectedScenario int) {
	t.gameData = gameData
	t.scenarioData = scenarioData
	t.selectedScenario = selectedScenario
	t.side0GeneralCount = 0
	for _, general := range t.scenarioData.Generals[0] {
		if general.Name == "" {
			break
		}
		t.side0GeneralCount++
	}
	t.side1GeneralCount = 0
	for _, general := range t.scenarioData.Generals[1] {
		if general.Name == "" {
			break
		}
		t.side1GeneralCount++
	}
	t.perGeneralTable.SetColumnCount(t.side0GeneralCount + t.side1GeneralCount + 1)
}

var perUnitFieldNames []string = []string{
	"Data0Low",
	"Data0High",
	"Data16Low",
	"Data16High",
	"Data32_8",
	"Data32_32",
	"Data32_64",
	"Data32_128",
	"AttackRange",
	"UnitScores",
	"RecoveryRate",
	"UnitMask0",
	"UnitMask1",
	"UnitMask2",
	"UnitUsesSupplies",
	"UnitMask4",
	"UnitMask5",
	"UnitCanMove",
	"UnitMask7",
	"Data200Low",
	"UnitResupply",
	"MenCountLimit",
	"TankCountLimit",
}
var perTerrainFieldNames []string = []string{
	"MenAttack",
	"TankAttack",
	"MenDefence",
	"TankDefence",
}
var perFormationFieldNames []string = []string{
	"MenAttack",
	"TankAttack",
	"MenDefence",
	"TankDefence",
	"Data192",
	"ChangeSpeed0",
	"ChangeSpeed1",
}
var perGeneralFieldNames []string = []string{
	"Data0_0",
	"Data0_1",
	"Data0_2",
	"Data0_3",
	"Data0_4",
	"Data0_5",
	"Data0_6",
	"Data0_7",
	"Attack",
	"Data1High",
	"Defence",
	"Data2High",
	"Movement",
}

func (t *InfoTable) perUnitFieldName(row int) string {
	if row < 0 || row >= len(perUnitFieldNames) {
		return ""
	}
	return perUnitFieldNames[row]
}
func (t *InfoTable) perTerrainFieldName(row int) string {
	if row < 0 || row >= len(perTerrainFieldNames) {
		return ""
	}
	return perTerrainFieldNames[row]
}
func (t *InfoTable) perFormationFieldName(row int) string {
	if row < 0 || row >= len(perFormationFieldNames) {
		return ""
	}
	return perFormationFieldNames[row]
}
func (t *InfoTable) perGeneralFieldName(row int) string {
	if row < 0 || row >= len(perGeneralFieldNames) {
		return ""
	}
	return perGeneralFieldNames[row]
}

func arrayIntFieldToString(arr []int, column int) string {
	if column < 0 || column >= len(arr) {
		return "?"
	}
	return intToString(arr[column])
}
func intToString(v int) string {
	return strconv.FormatInt(int64(v), 10)
}
func arrayBoolFieldToString(arr []bool, column int) string {
	if column < 0 || column >= len(arr) {
		return "?"
	}
	return boolToString(arr[column])
}
func boolToString(v bool) string {
	if v {
		return "T"
	} else {
		return "F"
	}
}

func (t *InfoTable) perUnitFieldValue(row, column int) string {
	if row < 0 || row >= len(perUnitFieldNames) {
		return ""
	}
	switch perUnitFieldNames[row] {
	case "Data0Low":
		return arrayIntFieldToString(t.scenarioData.Data.Data0Low[:], column)
	case "Data0High":
		return arrayIntFieldToString(t.scenarioData.Data.Data0High[:], column)
	case "Data16Low":
		return arrayIntFieldToString(t.scenarioData.Data.Data16Low[:], column)
	case "Data16High":
		return arrayIntFieldToString(t.scenarioData.Data.Data16High[:], column)
	case "Data32_8":
		return arrayBoolFieldToString(t.scenarioData.Data.Data32_8[:], column)
	case "Data32_32":
		return arrayBoolFieldToString(t.scenarioData.Data.Data32_32[:], column)
	case "Data32_64":
		return arrayBoolFieldToString(t.scenarioData.Data.Data32_64[:], column)
	case "Data32_128":
		return arrayBoolFieldToString(t.scenarioData.Data.Data32_128[:], column)
	case "AttackRange":
		return arrayIntFieldToString(t.scenarioData.Data.AttackRange[:], column)
	case "UnitScores":
		return arrayIntFieldToString(t.scenarioData.Data.UnitScores[:], column)
	case "RecoveryRate":
		return arrayIntFieldToString(t.scenarioData.Data.RecoveryRate[:], column)
	case "UnitMask0":
		return arrayBoolFieldToString(t.scenarioData.Data.UnitMask0[:], column)
	case "UnitMask1":
		return arrayBoolFieldToString(t.scenarioData.Data.UnitMask1[:], column)
	case "UnitMask2":
		return arrayBoolFieldToString(t.scenarioData.Data.UnitMask2[:], column)
	case "UnitUsesSupplies":
		return arrayBoolFieldToString(t.scenarioData.Data.UnitUsesSupplies[:], column)
	case "UnitMask4":
		return arrayBoolFieldToString(t.scenarioData.Data.UnitMask4[:], column)
	case "UnitMask5":
		return arrayBoolFieldToString(t.scenarioData.Data.UnitMask5[:], column)
	case "UnitCanMove":
		return arrayBoolFieldToString(t.scenarioData.Data.UnitCanMove[:], column)
	case "UnitMask7":
		return arrayBoolFieldToString(t.scenarioData.Data.UnitMask7[:], column)
	case "Data200Low":
		return arrayIntFieldToString(t.scenarioData.Data.Data200Low[:], column)
	case "UnitResupply":
		return arrayIntFieldToString(t.scenarioData.Data.UnitResupplyPerType[:], column)
	case "MenCountLimit":
		return arrayIntFieldToString(t.scenarioData.Data.MenCountLimit[:], column)
	case "TankCountLimit":
		return arrayIntFieldToString(t.scenarioData.Data.TankCountLimit[:], column)
	}
	return ""
}

func (t *InfoTable) perTerrainFieldValue(row, column int) string {
	if row < 0 || row >= len(perTerrainFieldNames) {
		return ""
	}
	switch perTerrainFieldNames[row] {
	case "MenAttack":
		return arrayIntFieldToString(t.scenarioData.Data.TerrainMenAttack[:], column)
	case "TankAttack":
		return arrayIntFieldToString(t.scenarioData.Data.TerrainTankAttack[:], column)
	case "MenDefence":
		return arrayIntFieldToString(t.scenarioData.Data.TerrainMenDefence[:], column)
	case "TankDefence":
		return arrayIntFieldToString(t.scenarioData.Data.TerrainTankDefence[:], column)
	}
	return ""
}

func (t *InfoTable) perFormationFieldValue(row, column int) string {
	if row < 0 || row >= len(perFormationFieldNames) {
		return ""
	}
	switch perFormationFieldNames[row] {
	case "MenAttack":
		return arrayIntFieldToString(t.scenarioData.Data.FormationMenAttack[:], column)
	case "TankAttack":
		return arrayIntFieldToString(t.scenarioData.Data.FormationTankAttack[:], column)
	case "MenDefence":
		return arrayIntFieldToString(t.scenarioData.Data.FormationMenDefence[:], column)
	case "TankDefence":
		return arrayIntFieldToString(t.scenarioData.Data.FormationTankDefence[:], column)
	case "Data192":
		return arrayIntFieldToString(t.scenarioData.Data.Data192[:], column)
	case "ChangeSpeed0":
		return arrayIntFieldToString(t.scenarioData.Data.FormationChangeSpeed[0][:], column)
	case "ChangeSpeed1":
		return arrayIntFieldToString(t.scenarioData.Data.FormationChangeSpeed[1][:], column)
	}
	return ""
}

func (t *InfoTable) perGeneralFieldValue(row, column int) string {
	if row < 0 || row >= len(perGeneralFieldNames) {
		return ""
	}
	if column < 0 || column >= t.side0GeneralCount + t.side1GeneralCount {
		return ""
	}
	var general *lib.General
	if column < t.side0GeneralCount {
		general = &t.scenarioData.Generals[0][column]
	} else {
		general = &t.scenarioData.Generals[1][column-t.side0GeneralCount]
	}
	switch perGeneralFieldNames[row] {
	case "Data0_0":
		return boolToString(general.Data0_0)
	case "Data0_1":
		return boolToString(general.Data0_1)
	case "Data0_2":
		return boolToString(general.Data0_2)
	case "Data0_3":
		return boolToString(general.Data0_3)
	case "Data0_4":
		return boolToString(general.Data0_4)
	case "Data0_5":
		return boolToString(general.Data0_5)
	case "Data0_6":
		return boolToString(general.Data0_6)
	case "Data0_7":
		return boolToString(general.Data0_7)
	case "Attack":
		return intToString(general.Attack)
	case "Data1High":
		return intToString(general.Data1High)
	case "Defence":
		return intToString(general.Defence)
	case "Data2High":
		return intToString(general.Data2High)
	case "Movement":
		return intToString(general.Movement)
	}
	return ""
}

func (t *InfoTable) perUnitDrawCallback(context fltk.TableContext, row, column, x, y, w, h int) {
	if t.scenarioData == nil {
		return
	}
	switch context {
	case fltk.ContextCell:
		if row < 0 || row >= len(perUnitFieldNames) {
			return
		}
		fltk.DrawBox(fltk.THIN_UP_BOX, x, y, w, h, fltk.WHITE)
		fltk.PushClip(x, y, w, h)
		fltk.SetDrawColor(fltk.BLACK)
		if column == 0 {
			fltk.Draw(t.perUnitFieldName(row), x, y, w, h, fltk.ALIGN_LEFT)
		} else {
			text := t.perUnitFieldValue(row, column-1)
			fltk.Draw(text, x, y, w, h, fltk.ALIGN_RIGHT)
		}
		fltk.PopClip()
	case fltk.ContextColHeader:
		fltk.DrawBox(fltk.UP_BOX, x, y, w, h, 0x8f8f8fff)
		fltk.SetDrawColor(fltk.BLACK)
		if column <= 0 || column > len(t.scenarioData.Data.UnitTypes) {
			return
		}
		fltk.PushClip(x, y, w, h)
		fltk.Draw(t.scenarioData.Data.UnitTypes[column-1], x, y, w, h, fltk.ALIGN_LEFT)
		fltk.PopClip()
	}
}
func (t *InfoTable) perTerrainDrawCallback(context fltk.TableContext, row, column, x, y, w, h int) {
	if t.scenarioData == nil {
		return
	}
	switch context {
	case fltk.ContextCell:
		if row < 0 || row >= len(perTerrainFieldNames) {
			return
		}
		fltk.DrawBox(fltk.THIN_UP_BOX, x, y, w, h, fltk.WHITE)
		fltk.PushClip(x, y, w, h)
		fltk.SetDrawColor(fltk.BLACK)
		if column == 0 {
			fltk.Draw(t.perTerrainFieldName(row), x, y, w, h, fltk.ALIGN_LEFT)
		} else {
			text := t.perTerrainFieldValue(row, column-1)
			fltk.Draw(text, x, y, w, h, fltk.ALIGN_RIGHT)
		}
		fltk.PopClip()
	case fltk.ContextColHeader:
		fltk.DrawBox(fltk.UP_BOX, x, y, w, h, 0x8f8f8fff)
		fltk.SetDrawColor(fltk.BLACK)
	}
}

func (t *InfoTable) perFormationDrawCallback(context fltk.TableContext, row, column, x, y, w, h int) {
	if t.scenarioData == nil {
		return
	}
	switch context {
	case fltk.ContextCell:
		if row < 0 || row >= len(perFormationFieldNames) {
			return
		}
		fltk.DrawBox(fltk.THIN_UP_BOX, x, y, w, h, fltk.WHITE)
		fltk.PushClip(x, y, w, h)
		fltk.SetDrawColor(fltk.BLACK)
		if column == 0 {
			fltk.Draw(t.perFormationFieldName(row), x, y, w, h, fltk.ALIGN_LEFT)
		} else {
			text := t.perFormationFieldValue(row, column-1)
			fltk.Draw(text, x, y, w, h, fltk.ALIGN_RIGHT)
		}
		fltk.PopClip()
	case fltk.ContextColHeader:
		fltk.DrawBox(fltk.UP_BOX, x, y, w, h, 0x8f8f8fff)
		fltk.SetDrawColor(fltk.BLACK)
		if column <= 0 || column > len(t.scenarioData.Data.Formations) {
			return
		}
		fltk.PushClip(x, y, w, h)
		fltk.Draw(t.scenarioData.Data.Formations[column-1], x, y, w, h, fltk.ALIGN_LEFT)
		fltk.PopClip()
	}
}

func (t *InfoTable) perGeneralDrawCallback(context fltk.TableContext, row, column, x, y, w, h int) {
	if t.scenarioData == nil {
		return
	}
	switch context {
	case fltk.ContextCell:
		if row < 0 || row >= len(perGeneralFieldNames) {
			return
		}
		fltk.DrawBox(fltk.THIN_UP_BOX, x, y, w, h, fltk.WHITE)
		fltk.PushClip(x, y, w, h)
		fltk.SetDrawColor(fltk.BLACK)
		if column == 0 {
			fltk.Draw(t.perGeneralFieldName(row), x, y, w, h, fltk.ALIGN_LEFT)
		} else {
			text := t.perGeneralFieldValue(row, column-1)
			fltk.Draw(text, x, y, w, h, fltk.ALIGN_RIGHT)
		}
		fltk.PopClip()
	case fltk.ContextColHeader:
		fltk.DrawBox(fltk.UP_BOX, x, y, w, h, 0x8f8f8fff)
		fltk.SetDrawColor(fltk.BLACK)
		if column <= 0 || column > t.side0GeneralCount + t.side1GeneralCount {
			return
		}
		fltk.PushClip(x, y, w, h)
		var text string
		if column-1 < t.side0GeneralCount {
			text = t.scenarioData.Generals[0][column-1].Name
		} else {
			text = t.scenarioData.Generals[1][column-1-t.side0GeneralCount].Name
		}
		fltk.Draw(text, x, y, w, h, fltk.ALIGN_LEFT)
		fltk.PopClip()
	}
}
