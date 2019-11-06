/*
 * MinIO Cloud Storage (C) 2016, 2018 MinIO, Inc.
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
import { connect } from "react-redux"
import classNames from "classnames"
import logo from "../../img/logo.svg"
import Alert from "../alert/Alert"
import * as actionsAlert from "../alert/actions"
import InputGroup from "./InputGroup"
import web from "../web"
import { Redirect } from "react-router-dom"
import { withI18n } from "react-i18next";






export class Login extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      accessKey: "",
      secretKey: "",
      value: localStorage.getItem('language')
    }
  }

  // Handle field changes
  accessKeyChange(e) {
    this.setState({
      accessKey: e.target.value
    })
  }

  secretKeyChange(e) {
    this.setState({
      secretKey: e.target.value
    })
  }

  handleSubmit(event) {
    event.preventDefault()
    const { showAlert, clearAlert, history } = this.props
    let message = ""
    if (this.state.accessKey === "") {
      message = "Access Key cannot be empty"
    }
    if (this.state.secretKey === "") {
      message = "Secret Key cannot be empty"
    }
    if (message) {
      showAlert("danger", message)
      return
    }
    web
      .Login({
        username: this.state.accessKey,
        password: this.state.secretKey
      })
      .then(res => {
        // Clear alerts from previous login attempts
        clearAlert()

        history.push("/")
      })
      .catch(e => {
        showAlert("danger", e.message)
      })
  }

  componentWillMount() {
    const { clearAlert } = this.props
    // Clear out any stale message in the alert of previous page
    clearAlert()
    document.body.classList.add("is-guest")
  }

  componentWillUnmount() {
    document.body.classList.remove("is-guest")
  }
  // changelang() {
  //   console.log('---------')
  //   const { t, i18n } = this.props;
  //   i18n.changeLanguage('zhch');
  //   this.forceUpdate();

  // }
  changeLang(event) {
    this.setState({value: event.target.value});
    localStorage.setItem('language', event.target.value);
    window.location.reload();
  }
  render() {
    const { clearAlert, alert } = this.props
    const { t, i18n } = this.props;

    if (web.LoggedIn()) {
      return <Redirect to={"/"} />
    }
    let alertBox = <Alert {...alert} onDismiss={clearAlert} />
    // Make sure you don't show a fading out alert box on the initial web-page load.
    if (!alert.message) alertBox = ""
    return (
      <div className="login">
        {alertBox}
        <div className="l-wrap">
          <form onSubmit={this.handleSubmit.bind(this)}>
            {/* {t('WelcometoReact')} */}
            <InputGroup
              value={this.state.accessKey}
              onChange={this.accessKeyChange.bind(this)}
              className="ig-dark"
              label={t('accessKey')}
              id="accessKey"
              name="username"
              type="text"
              spellCheck="false"
              required="required"
              autoComplete="username"
            />
            <InputGroup
              value={this.state.secretKey}
              onChange={this.secretKeyChange.bind(this)}
              className="ig-dark"
              label={t('secretKey')}
              id="secretKey"
              name="password"
              type="password"
              spellCheck="false"
              required="required"
              autoComplete="new-password"
            />
            <button className="lw-btn" type="submit">
              <i className="fas fa-sign-in-alt" />
            </button>
            <p className="white">{t('accessKey')}: lambda</p>
            <p className="white">{t('secretKey')}: 12345678</p>
          </form>
          <br />
         { t('language') }&nbsp;&nbsp;&nbsp;
          <select id="pid" onChange={this.changeLang.bind(this)} value={this.state.value}>
            <option value="en">English</option>
            <option value="zh_cn">中文</option>
          </select>
        </div>
        <div className="l-footer">
          <a className="lf-logo" href="https://www.lambdastorage.com/" target="_blank">
            < img src={logo} alt="" />
          </ a>
          <div className="lf-server">{window.location.host}</div>
        </div>
      </div>
    )
  }
}

const mapDispatchToProps = dispatch => {
  return {
    showAlert: (type, message) =>
      dispatch(actionsAlert.set({ type: type, message: message })),
    clearAlert: () => dispatch(actionsAlert.clear())
  }
}

export default connect(
  state => state,
  mapDispatchToProps
)(withI18n()(Login))