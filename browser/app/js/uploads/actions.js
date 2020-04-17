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

import Moment from "moment"
import storage from "local-storage-fallback"
import * as alertActions from "../alert/actions"
import * as objectsActions from "../objects/actions"
import * as actionsCommon from "../browser/actions"
import { getCurrentBucket } from "../buckets/selectors"
import { getCurrentPrefix } from "../objects/selectors"
import { minioBrowserPrefix } from "../constants"
import i18next from 'i18next';
import * as bucketActions from "../buckets/actions"
export const ADD = "uploads/ADD"
export const UPDATE_PROGRESS = "uploads/UPDATE_PROGRESS"
export const STOP = "uploads/STOP"
export const SHOW_ABORT_MODAL = "uploads/SHOW_ABORT_MODAL"

export const add = (slug, size, name) => ({
  type: ADD,
  slug,
  size,
  name
})

export const updateProgress = (slug, loaded) => ({
  type: UPDATE_PROGRESS,
  slug,
  loaded
})

export const stop = slug => ({
  type: STOP,
  slug
})

export const showAbortModal = () => ({
  type: SHOW_ABORT_MODAL,
  show: true
})

export const hideAbortModal = () => ({
  type: SHOW_ABORT_MODAL,
  show: false
})

let requests = {}

export const addUpload = (xhr, slug, size, name) => {
  return function (dispatch) {
    requests[slug] = xhr
    dispatch(add(slug, size, name))
  }
}

export const abortUpload = slug => {
  document.querySelector(".page-load").classList.remove("pl-5")
  document.querySelector(".page-load").classList.add("pl-0", "pl-1")
  return function (dispatch) {
    const xhr = requests[slug]
    if (xhr) {
      xhr.abort()
    }
    dispatch(stop(slug))
    dispatch(hideAbortModal())
  }
}

export const uploadFile = file => {
  return function (dispatch, getState) {
    const state = getState()
    const currentBucket = getCurrentBucket(state)
    if (!currentBucket) {
      dispatch(
        alertActions.set({
          type: "danger",
          message: `${i18next.t('n1')}`
        })
      )
      return
    }
    const currentPrefix = getCurrentPrefix(state)
    const objectName = `${currentPrefix}${file.name}`
    const uploadUrl = `${
      window.location.origin
      }${minioBrowserPrefix}/upload/${currentBucket}/${objectName}`
    const slug = `${currentBucket}-${currentPrefix}-${file.name}`

    let xhr = new XMLHttpRequest()
    xhr.open("PUT", uploadUrl, true)
    xhr.withCredentials = false
    const token = storage.getItem("token")
    if (token) {
      xhr.setRequestHeader(
        "Authorization",
        "Bearer " + storage.getItem("token")
      )
    }
    xhr.setRequestHeader(
      "x-amz-date",
      Moment()
        .utc()
        .format("YYYYMMDDTHHmmss") + "Z"
    )

    dispatch(addUpload(xhr, slug, file.size, file.name))

    xhr.onload = function (event) {
      if (xhr.status == 401 || xhr.status == 403) {
        document.querySelector(".page-load").classList.remove("pl-5")
        document.querySelector(".page-load").classList.add("pl-0", "pl-1")
        dispatch(hideAbortModal())
        dispatch(stop(slug))
        dispatch(
          alertActions.set({
            type: "danger",
            message: `${i18next.t('n3')}`
          })
        )
        // setTimeout(() => {
        //   window.location.reload ();
        // }, 5000);
      }
      if (xhr.status == 500 || xhr.status == 504) {
        document.querySelector(".page-load").classList.remove("pl-5")
        document.querySelector(".page-load").classList.add("pl-0", "pl-1")
        dispatch(hideAbortModal())
        dispatch(stop(slug))
        dispatch(
          alertActions.set({
            type: "danger",
            message: xhr.responseText
          })
        )
        // setTimeout(() => {
        //   window.location.reload ();
        // }, 5000);
      }
      if (xhr.status == 200) {
        document.querySelector(".page-load").classList.remove("pl-5")
        document.querySelector(".page-load").classList.add("pl-0", "pl-1")
        dispatch(hideAbortModal())
        dispatch(stop(slug))
        dispatch(
          alertActions.set({
            type: "success",
            message: `${i18next.t('object')} ${file.name} ${i18next.t('n2')}`
          })
        )
        dispatch(bucketActions.fetchBuckets())
        dispatch(objectsActions.selectPrefix(currentPrefix))
        dispatch(actionsCommon.fetchStorageInfo())
      }
    }

    xhr.upload.addEventListener("error", event => {
      document.querySelector(".page-load").classList.remove("pl-5")
      document.querySelector(".page-load").classList.add("pl-0", "pl-1")
      dispatch(stop(slug))
      dispatch(
        alertActions.set({
          type: "danger",
          message: `${i18next.t('n4')} ${file.name} `
        })
      )
      // setTimeout(() => {
      //   window.location.reload ();
      // }, 5000);
    })

    xhr.upload.addEventListener("progress", event => {
      if (event.lengthComputable) {
        let loaded = event.loaded
        let total = event.total
        // Update the counter
        dispatch(updateProgress(slug, loaded))
      }
    })

    xhr.send(file)
  }
}
