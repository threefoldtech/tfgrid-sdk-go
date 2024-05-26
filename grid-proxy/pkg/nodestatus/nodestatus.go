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

var (
	nilPower = "node.power IS NULL"

	poweredOn   = "node.power->> 'state' = 'Up' AND node.power->> 'target' = 'Up'"
	poweredOff  = "node.power->> 'state' = 'Down' AND node.power->> 'target' = 'Down'"
	poweringOff = "node.power->> 'state' = 'Up' AND node.power->> 'target' = 'Down'"
	poweringOn  = "node.power->> 'state' = 'Down' AND node.power->> 'target' = 'Up'"
)

// return the condition to be used in the SQL query to get the nodes with the given status.
func DecideNodeStatusCondition(statuses []string) string {
	nodeUpInterval := time.Now().Unix() - int64(nodeUpStateFactor)*int64(nodeUpReportInterval.Seconds())
	nodeStandbyInterval := time.Now().Unix() - int64(nodeStandbyStateFactor)*int64(nodeStandbyReportInterval.Seconds())

	inUpInterval := fmt.Sprintf("node.updated_at >= %d", nodeUpInterval)
	outUpInterval := fmt.Sprintf("node.updated_at < %d", nodeUpInterval)
	inStandbyInterval := fmt.Sprintf("node.updated_at >= %d", nodeStandbyInterval)
	outStandbyInterval := fmt.Sprintf("node.updated_at < %d", nodeStandbyInterval)

	condition := ""
	conditions := map[string]string{
		"up":      fmt.Sprintf(`%s AND (%s OR (%s))`, inUpInterval, nilPower, poweredOn),
		"down":    fmt.Sprintf(`(%s AND (%s OR (%s))) OR %s`, outUpInterval, nilPower, poweredOn, outStandbyInterval),
		"standby": fmt.Sprintf(`((%s) OR (%s) OR (%s)) AND %s`, poweredOff, poweringOff, poweringOn, inStandbyInterval),
	}

	for idx, status := range statuses {
		if idx != 0 && idx < len(statuses) {
			condition += " OR "
		}
		condition += "(" + conditions[status] + ")"
	}

	return condition
}

// DecideNodeStatusOrdering returns an sql ordering condition
func DecideNodeStatusOrdering(order types.SortOrder) string {
	nodeUpInterval := time.Now().Unix() - int64(nodeUpStateFactor)*int64(nodeUpReportInterval.Seconds())
	nodeStandbyInterval := time.Now().Unix() - int64(nodeStandbyStateFactor)*int64(nodeStandbyReportInterval.Seconds())

	inUpInterval := fmt.Sprintf("node.updated_at >= %d", nodeUpInterval)
	outUpInterval := fmt.Sprintf("node.updated_at < %d", nodeUpInterval)
	inStandbyInterval := fmt.Sprintf("node.updated_at >= %d", nodeStandbyInterval)
	outStandbyInterval := fmt.Sprintf("node.updated_at < %d", nodeStandbyInterval)

	upNodesOrder := 1
	standbyNodesOrder := 2
	downNodesOrder := 3

	if order == types.SortOrderDesc {
		upNodesOrder = 3
		downNodesOrder = 1
	}

	orderBy := "CASE "
	orderBy += fmt.Sprintf(`WHEN %s AND (%s OR (%s)) THEN %d `, inUpInterval, nilPower, poweredOn, upNodesOrder)
	orderBy += fmt.Sprintf(`WHEN ((%s) OR (%s) OR (%s)) AND %s THEN %d `, poweredOff, poweringOff, poweringOn, inStandbyInterval, standbyNodesOrder)
	orderBy += fmt.Sprintf(`WHEN (%s AND (%s OR (%s))) OR %s THEN %d `, outUpInterval, nilPower, poweredOn, outStandbyInterval, downNodesOrder)
	orderBy += "ELSE 4 END "
	orderBy += ", node.node_id "

	return orderBy
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
