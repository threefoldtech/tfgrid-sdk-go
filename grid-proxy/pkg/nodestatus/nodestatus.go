package nodestatus

import (
	"fmt"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

const (
	nodeUpStateFactor         = 2                // number of the cycles for the upInterval
	nodeUpReportInterval      = time.Minute * 40 // the interval to report for the up node
	nodeStandbyStateFactor    = 1                // number of the cycles for the standbyInterval
	nodeStandbyReportInterval = time.Hour * 24   // the interval to report for the standby node
)

// return the condition to be used in the SQL query to get the nodes with the given status.
func DecideNodeStatusCondition(status string) string {
	condition := "TRUE"

	nilPower := "node.power->> 'state' = '' AND node.power->> 'target' = ''"

	poweredOn := "node.power->> 'state' = 'Up' AND node.power->> 'target' = 'Up'"
	poweredOff := "node.power->> 'state' = 'Down' AND node.power->> 'target' = 'Down'"
	poweringOff := "node.power->> 'state' = 'Up' AND node.power->> 'target' = 'Down'"
	poweringOn := "node.power->> 'state' = 'Down' AND node.power->> 'target' = 'Up'"

	nodeUpInterval := time.Now().Unix() - int64(nodeUpStateFactor)*int64(nodeUpReportInterval.Seconds())
	nodeStandbyInterval := time.Now().Unix() - int64(nodeStandbyStateFactor)*int64(nodeStandbyReportInterval.Seconds())

	inUpInterval := fmt.Sprintf("node.updated_at >= %d", nodeUpInterval)
	outUpInterval := fmt.Sprintf("node.updated_at < %d", nodeUpInterval)
	inStandbyInterval := fmt.Sprintf("node.updated_at >= %d", nodeStandbyInterval)
	outStandbyInterval := fmt.Sprintf("node.updated_at < %d", nodeStandbyInterval)

	if status == "up" {
		condition = fmt.Sprintf(`%s AND (%s OR (%s))`, inUpInterval, nilPower, poweredOn)
	} else if status == "down" {
		condition = fmt.Sprintf(`(%s AND (%s OR (%s))) OR %s`, outUpInterval, nilPower, poweredOn, outStandbyInterval)
	} else if status == "standby" {
		condition = fmt.Sprintf(`((%s) OR (%s) OR (%s)) AND %s`, poweredOff, poweringOff, poweringOn, inStandbyInterval)
	}

	return condition
}

// return the status of the node based on the power status and the last update time.
func DecideNodeStatus(power types.NodePower, updatedAt int64) string {
	const down = "Down"
	const up = "Up"

	nilPower := power.State == "" && power.Target == ""
	poweredOff := power.State == down && power.Target == down
	poweredOn := power.State == up && power.Target == up
	poweringOn := power.State == down && power.Target == up
	poweringOff := power.State == up && power.Target == down

	nodeUpInterval := time.Now().Unix() - int64(nodeUpStateFactor)*int64(nodeUpReportInterval.Seconds())
	nodeStandbyInterval := time.Now().Unix() - int64(nodeStandbyStateFactor)*int64(nodeStandbyReportInterval.Seconds())

	inUpInterval := updatedAt >= nodeUpInterval
	inStandbyInterval := updatedAt >= nodeStandbyInterval

	if inUpInterval && (nilPower || poweredOn) {
		return "up"
	} else if (poweredOff || poweringOff || poweringOn) && inStandbyInterval {
		return "standby"
	} else {
		return "down"
	}
}
