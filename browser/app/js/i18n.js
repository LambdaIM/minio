import i18n from "i18next";
import { reactI18nextModule } from "react-i18next";

// the translations
// (tip move them in a JSON file and import them)
const resources = {
  en: {
    translation: {
      "WelcometoReact": "Welcome to React and react-i18next"
    }
  },
  zhch:{
    translation: {
        "WelcometoReact": "欢迎使用React and React-I18Next"
      }

  }
};

i18n
  .use(reactI18nextModule) // passes i18n down to react-i18next
  .init({
    resources,
    lng: "en",

    // keySeparator: false, // we do not use keys in form messages.welcome

    interpolation: {
      escapeValue: false // react already safes from xss
    }
  });

  export default i18n;