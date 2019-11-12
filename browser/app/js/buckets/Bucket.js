/*
 * MinIO Cloud Storage (C) 2018 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from "react"
import classNames from "classnames"
import { Popover, OverlayTrigger, ProgressBar } from "react-bootstrap"
import Moment from "moment"
import i18next from 'i18next';
export const Bucket = ({ bucket, isActive, selectBucket }) => {

  function getPercent(use, total) {
    return (use / total).toFixed(5) * 100;
  }
  function exTime() {
    if (bucket.expirationData == '0001-01-01T00:00:00Z') {
      return '--'
    }
    return Moment.unix(bucket.expirationData).format("YYYY-MM-DD HH:mm")
  }

  function createTime() {
    if (bucket.creationDate == '0001-01-01T00:00:00Z') {
      return '--'
    }
    return Moment.unix(bucket.creationDate).format("YYYY-MM-DD HH:mm")
  }

  const percent = getPercent(bucket.used, bucket.capacity) || 0;
  const popoverHoverFocus = (
    <Popover id="popover-trigger-hover-focus" title={i18next.t('info')}>
      {/* <strong>{bucket.name}</strong> */}
      <p>
        <span className="content-title">{i18next.t('nodeName')}</span>
        <br />
        <span className="content-value">{bucket.storageName}</span>
      </p>

      <p>
        <span className="content-title">{i18next.t('orderID')}</span>
        <br />
        <span className="content-value">
          <a href={`http://testbrowser.lambda.im/#/orderDetail/match/${bucket.name}`} target="_blank">
            {bucket.name}
          </a>
        </span>
      </p>

      <p>
        <span className="content-title">{i18next.t('sellerAddress')}</span>
        <br />
        <span className="content-value">
          <a href={`http://testbrowser.lambda.im/#/address/${bucket.sellerAddress}/activity/1/all`} target="_blank">
            {bucket.sellerAddress}
          </a>
        </span>
      </p>

      <div>
        <span className="content-title">{i18next.t('used')}  ({(bucket.used / 1073741824).toFixed(2)}/ {(bucket.capacity / 1073741824).toFixed(2)} GB)</span>
        <div>
          {
            percent <= 50 ? <ProgressBar label={`${percent.toFixed(2)}%`} striped className="progress-bar-success" now={percent} /> : <ProgressBar className="progress-bar-warning" label={`${percent.toFixed(2)}%`} bsStyle="warning" now={percent} />
          }
        </div>
      </div>


      <p>
        <span className="content-title">{i18next.t('createTime')}</span>
        <br />
        <span className="content-value">{createTime()}</span>
      </p>

      <p>
        <span className="content-title">{i18next.t('exTime')}</span>
        <br />
        <span className="content-value">{exTime()}</span>
      </p>
    </Popover>
  );
  return (
    // <Popover id="popover-positioned-right" title="Popover right">

    <OverlayTrigger
      trigger={['hover']}
      placement="right"
      overlay={popoverHoverFocus}
      rootClose={true}
      animation={false}
      delayHide={1000}
      delayShow={500}
    >
      <li
        className={classNames({
          active: isActive
        })}
        onClick={e => {
          e.preventDefault()
          selectBucket(bucket.name)
        }}
      >
        {
          bucket.status == 1 ? <i className="fa fa-check valid" aria-hidden="true"></i> : <i className="fa fa-close invalid" aria-hidden="true"></i>
        }
        <a
          href=""
          className={classNames({
            "fesli-loading": false
          })}
        >
          {bucket.name}
        </a>

      </li>
    </OverlayTrigger>


    // </Popover>
  )
}

export default Bucket
