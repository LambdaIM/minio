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
import { connect } from "react-redux"
import ConfirmModal from "../browser/ConfirmModal"
import * as uploadsActions from "./actions"
import { withI18n } from "react-i18next";
export class AbortConfirmModal extends React.Component {
  abortUploads() {
    const { abort, uploads } = this.props
    for (var slug in uploads) {
      abort(slug)
    }
  }
  render() {
    const { hideAbort } = this.props
    const { t, i18n } = this.props;
    let baseClass = classNames({
      "abort-upload": true
    })
    let okIcon = classNames({
      fas: true,
      "fa-times": true
    })
    let cancelIcon = classNames({
      fas: true,
      "fa-cloud-upload-alt": true
    })

    return (
      <ConfirmModal
        show={true}
        baseClass={baseClass}
        text={t('aboartText')}
        icon="fa fa-info-circle mci-amber"
        okText={t('abort')}
        okIcon={okIcon}
        cancelText={t('upload1')}
        cancelIcon={cancelIcon}
        okHandler={this.abortUploads.bind(this)}
        cancelHandler={hideAbort}
      />
    )
  }
}

const mapStateToProps = state => {
  return {
    uploads: state.uploads.files
  }
}

const mapDispatchToProps = dispatch => {
  return {
    abort: slug => dispatch(uploadsActions.abortUpload(slug)),
    hideAbort: () => dispatch(uploadsActions.hideAbortModal())
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(withI18n()(AbortConfirmModal))
