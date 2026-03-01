import Head from 'next/head';
import { useState } from 'react';

type Language = 'ru' | 'en';
type Tone = 'light' | 'dark' | 'gradient';

type LocalizedContent = {
  languageLabel: string;
  topbarCta: string;
  nav: {
    product: string;
    workflow: string;
    demo: string;
  };
  meta: {
    title: string;
    description: string;
  };
  hero: {
    eyebrow: string;
    title: string;
    subtitle: string;
    primaryCta: string;
    secondaryCta: string;
    deviceIncident: string;
    deviceTitle: string;
    deviceBody: string;
  };
  metrics: Array<{ label: string; value: string }>;
  intro: {
    title: string;
    body: string;
  };
  featureKicker: string;
  features: Array<{ title: string; body: string; tone: Tone }>;
  workflow: {
    kicker: string;
    title: string;
    steps: string[];
  };
  deploy: {
    kicker: string;
    title: string;
    body: string;
    cta: string;
  };
  quote: {
    body: string;
    author: string;
  };
  closeout: {
    title: string;
    repoButton: string;
    emailButton: string;
    contactTitle: string;
    repoTitle: string;
  };
};

const contactEmail = 'mikhail.harchuk@gmail.com';
const repoUrl = 'https://github.com/harchuk/runbook_hunter';

const content: Record<Language, LocalizedContent> = {
  ru: {
    languageLabel: 'Язык',
    topbarCta: 'Написать Михаилу',
    nav: {
      product: 'Продукт',
      workflow: 'Процесс',
      demo: 'Демо'
    },
    meta: {
      title: 'Runbook Hunter | Спокойный incident response без шума',
      description:
        'Runbook Hunter принимает алерты из Alertmanager, коррелирует инциденты, запускает безопасные диагностики и отправляет дедуплицированные обновления.'
    },
    hero: {
      eyebrow: 'Платформа Incident Intelligence',
      title: 'Меньше шума. Больше контроля.',
      subtitle:
        'Runbook Hunter превращает алерт-штормы в одну понятную историю инцидента. Команда получает фокус, безопасную диагностику и единый статус без хаоса.',
      primaryCta: 'Запросить демо',
      secondaryCta: 'Смотреть процесс',
      deviceIncident: 'Инцидент #4721',
      deviceTitle: 'Всплеск задержек API',
      deviceBody: '38 алертов объединены за 42 секунды'
    },
    metrics: [
      { label: 'Alertmanager', value: 'Нативный ingest' },
      { label: 'Каналы', value: 'Telegram + Mattermost' },
      { label: 'Runtime', value: 'Go, Next.js, Helm' },
      { label: 'Безопасность MVP', value: 'Только read-only' }
    ],
    intro: {
      title: 'Сделано для команд, которые масштабируют надежность, а не панику.',
      body:
        'Визуально и логически лендинг построен в продуктовой подаче: меньше информационного шума, больше ясности с первого экрана.'
    },
    featureKicker: 'Возможность',
    features: [
      {
        title: 'Один инцидент вместо пятидесяти алертов',
        body: 'Детерминированный fingerprinting и корреляция объединяют всплески в одну живую хронологию.',
        tone: 'light'
      },
      {
        title: 'Безопасная диагностика по умолчанию',
        body: 'Read-only runbook checks, allowlist, retry и timeout ускоряют анализ без рискованных write-операций.',
        tone: 'dark'
      },
      {
        title: 'Запуск в один Helm install',
        body: 'API, worker и UI разворачиваются как единый продукт. Начните с values.yaml и масштабируйтесь через DB overrides.',
        tone: 'gradient'
      }
    ],
    workflow: {
      kicker: 'Процесс',
      title: 'Три шага от шума к понятной картине.',
      steps: [
        'Принимаем алерты из Alertmanager вместе с labels и routing-контекстом.',
        'Коррелируем события в один непрерывный нарратив инцидента.',
        'Запускаем детерминированные read-only проверки и отправляем дедуплицированные апдейты.'
      ]
    },
    deploy: {
      kicker: 'Внедрение',
      title: 'Запуск в вашем кластере уже на этой неделе.',
      body:
        'Подключим Alertmanager, настроим runbook matching и подготовим production-ready конфигурацию без сложной миграции.',
      cta: 'Запросить архитектурный созвон'
    },
    quote: {
      body:
        '«Runbook Hunter убрал у нас петлю паники во время инцидентов. Мы наконец видим одну историю, а не 30 несвязанных алертов».',
      author: 'Platform Team Lead, SaaS Infrastructure'
    },
    closeout: {
      title: 'Интерфейс, который понятен даже в 03:00.',
      repoButton: 'Открыть GitHub репозиторий',
      emailButton: 'Написать на почту',
      contactTitle: 'Контакт',
      repoTitle: 'Репозиторий'
    }
  },
  en: {
    languageLabel: 'Language',
    topbarCta: 'Email Mikhail',
    nav: {
      product: 'Product',
      workflow: 'Workflow',
      demo: 'Demo'
    },
    meta: {
      title: 'Runbook Hunter | Calm incident response without alert chaos',
      description:
        'Runbook Hunter ingests Alertmanager alerts, correlates incidents, runs safe diagnostics, and sends deduplicated team updates.'
    },
    hero: {
      eyebrow: 'Incident Intelligence Platform',
      title: 'Calm signal. Faster action.',
      subtitle:
        'Runbook Hunter turns alert storms into one clear incident story. Your team gets focused context, safe diagnostics, and consistent updates.',
      primaryCta: 'Book a Demo',
      secondaryCta: 'See Workflow',
      deviceIncident: 'Incident #4721',
      deviceTitle: 'API Latency Burst',
      deviceBody: 'Correlated from 38 alerts in 42s'
    },
    metrics: [
      { label: 'Alertmanager', value: 'Native Ingest' },
      { label: 'Channels', value: 'Telegram + Mattermost' },
      { label: 'Runtime', value: 'Go, Next.js, Helm' },
      { label: 'MVP Safety', value: 'Read-only Execution' }
    ],
    intro: {
      title: 'Built for teams that scale reliability, not panic.',
      body:
        'The layout follows a product-first narrative: less dashboard noise, more clarity at a glance for on-call teams.'
    },
    featureKicker: 'Feature',
    features: [
      {
        title: 'See one incident, not fifty alerts',
        body: 'Deterministic fingerprinting and correlation merge noisy bursts into one live timeline.',
        tone: 'light'
      },
      {
        title: 'Diagnose safely by default',
        body: 'Read-only runbook checks, strict allowlists, retries, and timeouts keep investigations fast.',
        tone: 'dark'
      },
      {
        title: 'Launch in one Helm install',
        body: 'Deploy API, worker, and UI as one product. Start with values.yaml and scale with DB overrides.',
        tone: 'gradient'
      }
    ],
    workflow: {
      kicker: 'Workflow',
      title: 'Three steps from noise to narrative.',
      steps: [
        'Ingest alerts from Alertmanager with labels and routing context.',
        'Correlate events into one incident narrative in real time.',
        'Run deterministic read-only diagnostics and push deduplicated updates.'
      ]
    },
    deploy: {
      kicker: 'Deployment',
      title: 'Run in your cluster this week.',
      body:
        'Connect Alertmanager, tune runbook matching, and launch with production-minded defaults without disruptive migration.',
      cta: 'Request Architecture Call'
    },
    quote: {
      body:
        '"Runbook Hunter removed the panic loop from our incidents. We finally see one story instead of thirty disconnected alerts."',
      author: 'Platform Team Lead, SaaS Infrastructure'
    },
    closeout: {
      title: 'Built to feel obvious at 3:00 AM.',
      repoButton: 'Open GitHub Repository',
      emailButton: 'Send Email',
      contactTitle: 'Contact',
      repoTitle: 'Repository'
    }
  }
};

export default function Landing() {
  const [language, setLanguage] = useState<Language>('ru');
  const t = content[language];

  return (
    <>
      <Head>
        <title>{t.meta.title}</title>
        <meta name="description" content={t.meta.description} />
        <link
          rel="icon"
          href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'%3E%3Crect width='32' height='32' rx='8' fill='%230a84ff'/%3E%3Ctext x='16' y='21' text-anchor='middle' font-size='14' font-family='Arial' fill='white'%3ER%3C/text%3E%3C/svg%3E"
        />
      </Head>

      <main className="page">
        <header className="topbar">
          <a className="brand" href="/">Runbook Hunter</a>
          <nav className="topnav" aria-label="Main navigation">
            <a href="#product">{t.nav.product}</a>
            <a href="#workflow">{t.nav.workflow}</a>
            <a href="#demo">{t.nav.demo}</a>
          </nav>
          <div className="topbar-controls">
            <div className="language-switch" role="group" aria-label={t.languageLabel}>
              <button
                type="button"
                className={language === 'ru' ? 'lang-btn active' : 'lang-btn'}
                onClick={() => setLanguage('ru')}
              >
                RU
              </button>
              <button
                type="button"
                className={language === 'en' ? 'lang-btn active' : 'lang-btn'}
                onClick={() => setLanguage('en')}
              >
                EN
              </button>
            </div>
            <a className="topbar-cta" href={`mailto:${contactEmail}`}>{t.topbarCta}</a>
          </div>
        </header>

        <section className="hero">
          <p className="eyebrow">{t.hero.eyebrow}</p>
          <h1>{t.hero.title}</h1>
          <p className="subtitle">{t.hero.subtitle}</p>

          <div className="cta-row">
            <a className="btn primary" href="#demo">{t.hero.primaryCta}</a>
            <a className="btn" href="#workflow">{t.hero.secondaryCta}</a>
          </div>

          <div className="hero-device" aria-hidden="true">
            <div className="orb orb-one" />
            <div className="orb orb-two" />
            <div className="device-card">
              <span>{t.hero.deviceIncident}</span>
              <strong>{t.hero.deviceTitle}</strong>
              <p>{t.hero.deviceBody}</p>
            </div>
          </div>

          <div className="proof-grid">
            {t.metrics.map((item) => (
              <article key={item.label} className="proof-card">
                <span>{item.label}</span>
                <strong>{item.value}</strong>
              </article>
            ))}
          </div>
        </section>

        <section id="product" className="section intro">
          <h2>{t.intro.title}</h2>
          <p>{t.intro.body}</p>
        </section>

        <section className="cards">
          {t.features.map((card) => (
            <article key={card.title} className={`card ${card.tone}`}>
              <p className="card-kicker">{t.featureKicker}</p>
              <h3>{card.title}</h3>
              <p>{card.body}</p>
            </article>
          ))}
        </section>

        <section id="workflow" className="section split">
          <article className="workflow-card dark">
            <p className="card-kicker">{t.workflow.kicker}</p>
            <h3>{t.workflow.title}</h3>
            <ol>
              {t.workflow.steps.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ol>
          </article>

          <article id="demo" className="workflow-card light">
            <p className="card-kicker">{t.deploy.kicker}</p>
            <h3>{t.deploy.title}</h3>
            <p>{t.deploy.body}</p>
            <a className="btn primary" href={`mailto:${contactEmail}`}>{t.deploy.cta}</a>
          </article>
        </section>

        <section className="section quote">
          <p>{t.quote.body}</p>
          <span>{t.quote.author}</span>
        </section>

        <section className="section closeout">
          <h2>{t.closeout.title}</h2>
          <div className="cards">
            <a className="btn primary" href={repoUrl} target="_blank" rel="noreferrer">{t.closeout.repoButton}</a>
            <a className="btn" href={`mailto:${contactEmail}`}>{t.closeout.emailButton}</a>
          </div>
          <p className="info-line">
            {t.closeout.contactTitle}:{' '}
            <a href={`mailto:${contactEmail}`}>{contactEmail}</a>
          </p>
          <p className="info-line">
            {t.closeout.repoTitle}:{' '}
            <a href={repoUrl} target="_blank" rel="noreferrer">{repoUrl}</a>
          </p>
        </section>
      </main>
    </>
  );
}
