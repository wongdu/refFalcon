// Copyright 2017 Xiaomi, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package store

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/falcon-plus/modules/judge/g"
)

func Judge(L *SafeLinkedList, firstItem *model.JudgeItem, now int64) {
	CheckStrategy(L, firstItem, now)
	CheckExpression(L, firstItem, now)
}

func CheckStrategy(L *SafeLinkedList, firstItem *model.JudgeItem, now int64) {
	key := fmt.Sprintf("%s/%s", firstItem.Endpoint, firstItem.Metric)
	strategyMap := g.StrategyMap.Get()
	strategies, exists := strategyMap[key]
	if !exists {
		return
	}

	for _, s := range strategies {
		// 因为key仅仅是endpoint和metric，所以得到的strategies并不一定是与当前judgeItem相关的
		// 比如lg-dinp-docker01.bj配置了两个proc.num的策略，一个name=docker，一个name=agent
		// 所以此处要排除掉一部分
		related := true
		for tagKey, tagVal := range s.Tags {
			if myVal, exists := firstItem.Tags[tagKey]; !exists || myVal != tagVal {
				related = false
				break
			}
		}

		if !related {
			continue
		}

		judgeItemWithStrategy(L, s, firstItem, now)
	}
}

func judgeItemWithStrategy(L *SafeLinkedList, strategy model.Strategy, firstItem *model.JudgeItem, now int64) {
	fn, err := ParseFuncFromString(strategy.Func, strategy.Operator, strategy.RightValue)
	if err != nil {
		log.Printf("[ERROR] parse func %s fail: %v. strategy id: %d", strategy.Func, err, strategy.Id)
		return
	}

	//20200304_14:20：如果当前的value
	//在调用Compute接口前预先调用处理恢复通知的数据，同样的逻辑需要在judgeItemWithExpression函数中也要实现，
	//暂时不加，因为dashboard那里未配置expression策略，只配置了strategy策略。
	if priorityValue, exist := g.Config().PriorityClusters[firstItem.Metric]; exist && priorityValue == int(firstItem.Value) {
		key := fmt.Sprintf("%s_%s", firstItem.Endpoint, firstItem.Metric)
		priorityMetricEventMap := g.PriorityMetricEventMap.Get()
		priorityMetricEvents, exists := priorityMetricEventMap[key]
		if exists {
			for modelEventItem := range priorityMetricEvents {
				modelEventItem.Status = "DDOK"
				modelEventItem.CurrentStep = 1
				sendEventWithoutSave(modelEventItem)
			}

			g.PriorityMetricEventMap.Delete(key)
		}
	}

	historyData, leftValue, isTriggered, isEnough := fn.Compute(L)
	if !isEnough {
		return
	}

	event := &model.Event{
		Id:         fmt.Sprintf("s_%d_%s", strategy.Id, firstItem.PrimaryKey()),
		Strategy:   &strategy,
		Endpoint:   firstItem.Endpoint,
		LeftValue:  leftValue,
		EventTime:  firstItem.Timestamp,
		PushedTags: firstItem.Tags,
	}

	sendEventIfNeed(historyData, isTriggered, now, event, strategy.MaxStep, firstItem.Metric)
}

func sendEvent(event *model.Event, endpointMetric string) {
	// update last event
	g.LastEvents.Set(event.Id, event)

	if priorityValue, exist := g.Config().PriorityClusters[endpointMetric]; exist {
		key := fmt.Sprintf("%s_%s", event.Endpoint, endpointMetric)
		priorityMetricEventMap := g.PriorityMetricEventMap.Get()
		priorityMetricEvents, exists := priorityMetricEventMap[key]
		if exists {
			priorityMetricEvents = append(priorityMetricEvents, event)
		} else {
			priorityMetricEventMap[key] = []*model.Event{event}
		}
		g.PriorityMetricEventMap.ReInit(priorityMetricEventMap)
	}

	sendEventWithoutSave(event)
}

func sendEventWithoutSave(event *model.Event) {
	bs, err := json.Marshal(event)
	if err != nil {
		log.Printf("json marshal event %v fail: %v", event, err)
		return
	}

	// send to redis
	redisKey := fmt.Sprintf(g.Config().Alarm.QueuePattern, event.Priority())
	rc := g.RedisConnPool.Get()
	defer rc.Close()
	rc.Do("LPUSH", redisKey, string(bs))
}

func CheckExpression(L *SafeLinkedList, firstItem *model.JudgeItem, now int64) {
	keys := buildKeysFromMetricAndTags(firstItem)
	if len(keys) == 0 {
		return
	}

	// expression可能会被多次重复处理，用此数据结构保证只被处理一次
	handledExpression := make(map[int]struct{})

	expressionMap := g.ExpressionMap.Get()
	for _, key := range keys {
		expressions, exists := expressionMap[key]
		if !exists {
			continue
		}

		related := filterRelatedExpressions(expressions, firstItem)
		for _, exp := range related {
			if _, ok := handledExpression[exp.Id]; ok {
				continue
			}
			handledExpression[exp.Id] = struct{}{}
			judgeItemWithExpression(L, exp, firstItem, now)
		}
	}
}

func buildKeysFromMetricAndTags(item *model.JudgeItem) (keys []string) {
	for k, v := range item.Tags {
		keys = append(keys, fmt.Sprintf("%s/%s=%s", item.Metric, k, v))
	}
	keys = append(keys, fmt.Sprintf("%s/endpoint=%s", item.Metric, item.Endpoint))
	return
}

func filterRelatedExpressions(expressions []*model.Expression, firstItem *model.JudgeItem) []*model.Expression {
	size := len(expressions)
	if size == 0 {
		return []*model.Expression{}
	}

	exps := make([]*model.Expression, 0, size)

	for _, exp := range expressions {

		related := true

		itemTagsCopy := firstItem.Tags
		// 注意：exp.Tags 中可能会有一个endpoint=xxx的tag
		if _, ok := exp.Tags["endpoint"]; ok {
			itemTagsCopy = copyItemTags(firstItem)
		}

		for tagKey, tagVal := range exp.Tags {
			if myVal, exists := itemTagsCopy[tagKey]; !exists || myVal != tagVal {
				related = false
				break
			}
		}

		if !related {
			continue
		}

		exps = append(exps, exp)
	}

	return exps
}

func copyItemTags(item *model.JudgeItem) map[string]string {
	ret := make(map[string]string)
	ret["endpoint"] = item.Endpoint
	if item.Tags != nil && len(item.Tags) > 0 {
		for k, v := range item.Tags {
			ret[k] = v
		}
	}
	return ret
}

func judgeItemWithExpression(L *SafeLinkedList, expression *model.Expression, firstItem *model.JudgeItem, now int64) {
	fn, err := ParseFuncFromString(expression.Func, expression.Operator, expression.RightValue)
	if err != nil {
		log.Printf("[ERROR] parse func %s fail: %v. expression id: %d", expression.Func, err, expression.Id)
		return
	}

	historyData, leftValue, isTriggered, isEnough := fn.Compute(L)
	if !isEnough {
		return
	}

	event := &model.Event{
		Id:         fmt.Sprintf("e_%d_%s", expression.Id, firstItem.PrimaryKey()),
		Expression: expression,
		Endpoint:   firstItem.Endpoint,
		LeftValue:  leftValue,
		EventTime:  firstItem.Timestamp,
		PushedTags: firstItem.Tags,
	}

	sendEventIfNeed(historyData, isTriggered, now, event, expression.MaxStep, firstItem.Metric)
}

//20200304_14:47：增加metric，如果是PriorityCluster簇中的metric，则单独放到EndpointMetricEvents集合中
func sendEventIfNeed(historyData []*model.HistoryData, isTriggered bool, now int64, event *model.Event, maxStep int, endpointMetric string) {
	lastEvent, exists := g.LastEvents.Get(event.Id)
	if isTriggered {
		//检测当前metric如果是自定义的告警，就直接发送，不再保存做恢复判断，即ALERT
		bIgnore := false
		for _, v := range g.Config().IgnoreSelfMetrics {
			if v == event.Metric() {
				bIgnore = true
				break
			}
		}
		if bIgnore {
			event.Status = "DDALERT" //dingding_alert
			sendEventWithoutSave(event)
			return
		}

		event.Status = "PROBLEM"

		//自定义的错误告警，保存做恢复判断，即ALARM
		bProblemSelf := false
		for _, v := range g.Config().ProblemSelfMetrics {
			if v == event.Metric() {
				bProblemSelf = true
				break
			}
		}
		if bProblemSelf {
			event.Status = "DDALARM"
		}

		if !exists || lastEvent.Status[0] == 'O' || lastEvent.Status == "DDOK" {
			// 本次触发了阈值，之前又没报过警，得产生一个报警Event
			event.CurrentStep = 1

			// 但是有些用户把最大报警次数配置成了0，相当于屏蔽了，要检查一下
			if maxStep == 0 {
				return
			}

			sendEvent(event, endpointMetric)
			return
		}

		// 逻辑走到这里，说明之前Event是PROBLEM状态
		if lastEvent.CurrentStep >= maxStep {
			// 报警次数已经足够多，到达了最多报警次数了，不再报警
			return
		}

		if historyData[len(historyData)-1].Timestamp <= lastEvent.EventTime {
			// 产生过报警的点，就不能再使用来判断了，否则容易出现一分钟报一次的情况
			// 只需要拿最后一个historyData来做判断即可，因为它的时间最老
			return
		}

		if now-lastEvent.EventTime < g.Config().Alarm.MinInterval {
			// 报警不能太频繁，两次报警之间至少要间隔MinInterval秒，否则就不能报警
			return
		}

		event.CurrentStep = lastEvent.CurrentStep + 1
		sendEvent(event, endpointMetric)
	} else {
		// 如果LastEvent是Problem，报OK，否则啥都不做
		if exists {
			if lastEvent.Status[0] == 'P' {
				event.Status = "OK"
				event.CurrentStep = 1
				sendEvent(event, endpointMetric)
			} else if lastEvent.Status == "DDALARM" {
				event.Status = "DDOK"
				event.CurrentStep = 1
				sendEvent(event, endpointMetric)
			}
		}
		//2020-03-04_20:18：无需在此处理priority metric event，因为在收到恢复通知数据时已经处理了，
		//即在judgeItemWithStrategy函数中在调用func.go文件中的Compute接口之前
	}
}
