/*
 * Utility functions for quantifier package
 */

package quantifier

import (
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
)

func printLevelTable(levels *[]QuantificationLevel) {
	data := [][]string{}

	for _, level := range *levels {
		newlevel := []string{level.Name, strconv.Itoa(level.Start), strconv.Itoa(level.End)}
		data = append(data, newlevel)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Level", "From", "To"})
	table.SetBorder(true)  // Set Border to false
	table.AppendBulk(data) // Add Bulk Data
	table.Render()
}
