import { useTranslation } from '../lib/i18n'

export function AppHeader() {
  const { t, lang, setLang } = useTranslation()

  return (
    <header className="hero">
      <div className="heroContent">
        <p className="eyebrow">{t('header.tagline')}</p>
        <h1>{t('header.title')}</h1>
        <p className="heroCopy">{t('header.description')}</p>
      </div>
      <button
        type="button"
        className="langToggle"
        onClick={() => setLang(lang === 'en' ? 'zh' : 'en')}
        aria-label="Switch language"
      >
        {t('header.langSwitch')}
      </button>
    </header>
  )
}
