// Copyright © 2023 OpenIM. All rights reserved.
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

package controller

import (
	"context"

	"github.com/openimsdk/open-im-server/v3/pkg/common/config"
	localKafka "github.com/openimsdk/open-im-server/v3/pkg/common/kafka"
	"github.com/openimsdk/open-im-server/v3/pkg/common/storage/cache"
	"github.com/openimsdk/protocol/push"
	"github.com/openimsdk/protocol/sdkws"
	"github.com/openimsdk/tools/log"
)

type PushDatabase interface {
	DelFcmToken(ctx context.Context, userID string, platformID int) error
	MsgToOfflinePushMQ(ctx context.Context, key string, userIDs []string, msg2mq *sdkws.MsgData) error
}

type pushDataBase struct {
	cache                 cache.ThirdCache
	producerToOfflinePush localKafka.MessageProducer
}

func NewPushDatabase(cache cache.ThirdCache, kafkaConf *config.Kafka) (PushDatabase, error) {
	producerToOfflinePush, err := localKafka.NewMultiProducer(kafkaConf.BuildClusters(), kafkaConf.ToOfflinePushTopic)
	if err != nil {
		return nil, err
	}
	return &pushDataBase{
		cache:                 cache,
		producerToOfflinePush: producerToOfflinePush,
	}, nil
}

func (p *pushDataBase) DelFcmToken(ctx context.Context, userID string, platformID int) error {
	return p.cache.DelFcmToken(ctx, userID, platformID)
}

func (p *pushDataBase) MsgToOfflinePushMQ(ctx context.Context, key string, userIDs []string, msg2mq *sdkws.MsgData) error {
	_, _, err := p.producerToOfflinePush.SendMessage(ctx, key, &push.PushMsgReq{MsgData: msg2mq, UserIDs: userIDs})
	log.ZInfo(ctx, "message is push to offlinePush topic", "key", key, "userIDs", userIDs, "msg", msg2mq.String())
	return err
}
