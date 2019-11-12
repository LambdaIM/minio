import i18n from "i18next";
import { reactI18nextModule } from "react-i18next";

// the translations
// (tip move them in a JSON file and import them)
const resources = {
  en: {
    translation: {
      // "WelcometoReact": "Welcome to React and react-i18next"
      about: {
        "version": "Version",
        // ""
      },
      "accessKey": "Access Key",
      "secretKey": "Secret Key",
      "used": "Used",
      "language": "Language",
      "fullScreen": "FullScreen",
      "doc": "Documentation",
      "signout": "Sign Out",
      "search1": "Search Buckets...",
      "upload": "Drag the file to this area to upload the file immediately",
      "name": "File Name",
      "size": "Size",
      "modified": "Last Modified",
      "share": "Shareable Link",
      "expires": "Expires in (Max 7 days)",
      "copy": "Copy Link",
      "cancel": "Cancel",
      'copied': "Link copied to clipboard!",
      "object": "Object",
      "objects": "Objects",
      "select": "Selected",
      'delete': "Delete",
      "download": "Download",
      "zip": " all as zip",
      "deleteText": "Are you sure you want to delete?",
      "uploading": "Uploading",
      "aboartText": "Abort uploads in progress?",
      "abort": "Abort",
      "upload1": "Upload",
      "n1": "Please choose a bucket before trying to upload files.",
      "n2": "uploaded successfully",
      "n3": "Unauthorized request.",
      "n4": "Error occurred uploading ",
      "mutiFile": "Downloading multiple files is not supported",
      "info": 'Order Info',
      "nodeName": 'Storage Node Name',
      "orderID": "Order ID",
      "sellerAddress": "Seller Address",
      "createTime": "Create Time",
      "exTime": "Expire Time"
    }
  },
  zh_cn: {
    translation: {
      // "WelcometoReact": "欢迎使用React and React-I18Next"
      about: {
        "version": "版本",
        // ""
      },
      "accessKey": "密钥",
      "secretKey": "私钥",
      "used": "已用空间",
      "language": "切换语言",
      "fullScreen": "全屏",
      "doc": "文档",
      "signout": "退出",
      "search1": "搜索空间",
      "upload": "将文件拖到该区域以立即上传文件",
      "name": "文件名",
      "size": "文件大小",
      "modified": "最后更新时间",
      "share": "分享链接",
      "expires": "有效期 (最长7天)",
      "copy": "复制链接",
      "cancel": "取消",
      'copied': "链接复制成功",
      "object": "文件",
      "objects": "文件",
      'select': "被选择",
      'delete': "删除文件",
      "download": "下载",
      "zip": "所有",
      "deleteText": "你确定要删除这个文件吗?",
      "uploading": "上传中",
      "aboartText": "确定要中止上传文件吗?",
      "abort": "中止上传文件",
      "upload1": "继续上传",
      "n1": "在上传文件前请选择一个空间",
      "n2": "上传成功",
      "n3": "非法请求",
      "n4": "文件上传出错,出错文件:",
      "mutiFile": "不支持同时下载多个文件",
      "info": '订单信息',
      "nodeName": '节点名称',
      "orderID": "订单ID",
      "sellerAddress": "卖方地址",
      "createTime": "创建时间",
      "exTime": "过期时间"
    }

  }
};

export const getLanguage = () => {
  let language = localStorage.getItem('language');
  const lang = navigator.language || navigator.userLanguage; // 常规浏览器语言和IE浏览器
  language = language || lang;
  language = language.replace(/-/, '_').toLowerCase();
  if (language === 'zh_cn' || language === 'zh') {
    language = 'zh_cn';
  } else {
    language = 'en'
  }
  return language;
}

i18n
  .use(reactI18nextModule) // passes i18n down to react-i18next
  .init({
    resources,
    lng: getLanguage(),
    // keySeparator: false, // we do not use keys in form messages.welcome

    interpolation: {
      escapeValue: false // react already safes from xss
    }
  }, function (err, t) {
    // initialized and ready to go!
    document.getElementById('root').innerHTML = i18n.t('key');
  });

export default i18n;