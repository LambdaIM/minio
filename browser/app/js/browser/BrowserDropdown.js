/*
 * MinIO Cloud Storage (C) 2016, 2017, 2018 MinIO, Inc.
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
import { Dropdown } from "react-bootstrap"
import * as browserActions from "./actions"
import web from "../web"
import history from "../history"
import AboutModal from "./AboutModal"
import ChangePasswordModal from "./ChangePasswordModal"
import { Trans } from 'react-i18next';

export class BrowserDropdown extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      showAboutModal: false,
      showChangePasswordModal: false,
      value: localStorage.getItem('language')
    }
  }
  showAbout(e) {
    e.preventDefault()
    this.setState({
      showAboutModal: true
    })
  }
  hideAbout() {
    this.setState({
      showAboutModal: false
    })
  }
  changeLang(event) {
    this.setState({value: event.target.value});
    localStorage.setItem('language', event.target.value);
    window.location.reload();
  }

  showChangePassword(e) {
    e.preventDefault()
    this.setState({
      showChangePasswordModal: true
    })
  }
  hideChangePassword() {
    this.setState({
      showChangePasswordModal: false
    })
  }
  componentDidMount() {
    const { fetchServerInfo } = this.props
    fetchServerInfo()
  }
  fullScreen(e) {
    e.preventDefault()
    let el = document.documentElement
    if (el.requestFullscreen) {
      el.requestFullscreen()
    }
    if (el.mozRequestFullScreen) {
      el.mozRequestFullScreen()
    }
    if (el.webkitRequestFullscreen) {
      el.webkitRequestFullscreen()
    }
    if (el.msRequestFullscreen) {
      el.msRequestFullscreen()
    }
  }
  logout(e) {
    e.preventDefault()
    web.Logout()
    history.replace("/login")
  }

  render() {
    const { serverInfo } = this.props
    return (
      <li>
        <Dropdown pullRight id="top-right-menu">
          <Dropdown.Toggle noCaret>
            <i className="fas fa-bars" />
          </Dropdown.Toggle>
          <Dropdown.Menu className="dropdown-menu-right">
            <li>
              <a target="_blank" href="https://github.com/LambdaIM">
                GitHub <i className="fa fa-github" />
              </a>
            </li>
            <li>
              <a href="" onClick={this.fullScreen}>
              <Trans>fullScreen</Trans>  <i className="fa fa-expand" />
              </a>
            </li>
            <li>
              <a target="_blank" href="https://github.com/LambdaIM/launch">
              <Trans>doc</Trans>  <i className="fa fa-book" />
              </a>
            </li>
            {/* <li>
              <a target="_blank" href="https://slack.min.io">
                Ask for help <i className="fas fa-question-circle" />
              </a>
            </li> */}
            {/* <li>
              <a href="" id="show-about" onClick={this.showAbout.bind(this)}>
                About <i className="fas fa-info-circle" />
              </a>
              {this.state.showAboutModal && (
                <AboutModal
                  serverInfo={serverInfo}
                  hideAbout={this.hideAbout.bind(this)}
                />
              )}
            </li> */}
            {/* <li>
              <a href="" onClick={this.showChangePassword.bind(this)}>
                Change Password <i className="fas fa-cog" />
              </a>
              {this.state.showChangePasswordModal && (
                <ChangePasswordModal
                  serverInfo={serverInfo}
                  hideChangePassword={this.hideChangePassword.bind(this)}
                />
              )}
            </li> */}
            <li>
              {/* <a href="" id="logout" onClick={this.changeLang}>
                切换语言 <i className="fa fa-sign-out" />
              </a> */}
              <a href="javascript:void(0)">
              <Trans>language</Trans> &nbsp;
                <select id="pid" onChange={this.changeLang.bind(this)} value={this.state.value}>
                  <option value="en">English</option>
                  <option value="zh_cn">中文</option>
                </select>

              </a>

            </li>

            <li>
              <a href="" id="logout" onClick={this.logout}>
              <Trans>signout</Trans>  <i className="fa fa-sign-out" />
              </a>
            </li>
          </Dropdown.Menu>
        </Dropdown>
      </li>
    )
  }
}

const mapStateToProps = state => {
  return {
    serverInfo: state.browser.serverInfo
  }
}

const mapDispatchToProps = dispatch => {
  return {
    fetchServerInfo: () => dispatch(browserActions.fetchServerInfo())
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(BrowserDropdown)
