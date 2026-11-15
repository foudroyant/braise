(() => {
  "use strict";

  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));
  const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  /* ---------- année ---------- */

  const year = $("#year");
  if (year) year.textContent = String(new Date().getFullYear());

  /* ---------- téléchargement adapté à l'appareil ---------- */

  const heroDownload = $("#hero-download");
  const heroOS = $("#hero-os");

  const platform = (
    (navigator.userAgentData && navigator.userAgentData.platform) ||
    navigator.platform ||
    ""
  ).toLowerCase();
  const ua = navigator.userAgent || "";
  const isMobile = /android|iphone|ipad|ipod/i.test(ua);

  let target = null;

  if (isMobile) {
    if (heroOS) heroOS.textContent = "Sur téléphone, Braise s'ouvre dans le navigateur — rien à installer.";
  } else if (/win/.test(platform)) {
    target = 'a.os-row[data-key="windows-amd64"]';
    if (heroOS) heroOS.textContent = "Windows · x86_64 — téléchargement direct";
  } else if (/mac/.test(platform)) {
    target = 'a.os-row[data-group="mac"]';
    if (heroOS) heroOS.textContent = "macOS · Apple Silicon ou Intel — choisissez ci-dessous";
  } else if (/linux|x11/.test(platform)) {
    target = 'a.os-row[data-key="linux-amd64"]';
    if (heroOS) heroOS.textContent = "Linux · x86_64 — téléchargement direct";
  }

  if (target) {
    const rows = $$(target);
    rows.forEach((row) => row.classList.add("rec"));

    if (rows.length === 1 && heroDownload) {
      heroDownload.href = rows[0].href;
    }
  }

  /* ---------- topbar au défilement ---------- */

  const topbar = $(".topbar");
  const onScroll = () => {
    if (topbar) topbar.classList.toggle("is-scrolled", window.scrollY > 12);
  };
  onScroll();
  window.addEventListener("scroll", onScroll, { passive: true });

  /* ---------- animations ---------- */

  if (!window.gsap) return;

  if (window.ScrollTrigger) gsap.registerPlugin(ScrollTrigger);

  if (reduceMotion) {
    gsap.set(".hero h1 .line > span", { clearProps: "all" });
    return;
  }

  const hero = gsap.timeline({ defaults: { ease: "power3.out" } });

  hero
    .from(".hero-copy .eyebrow", { y: 16, autoAlpha: 0, duration: .55 })
    .from(".hero h1 .line > span", { yPercent: 112, duration: .95, stagger: .085 }, "-=.25")
    .from(".lede", { y: 18, autoAlpha: 0, duration: .65 }, "-=.55")
    .from(".cta, .cta-note", { y: 16, autoAlpha: 0, duration: .55, stagger: .08 }, "-=.4")
    .from(".specs", { y: 14, autoAlpha: 0, duration: .5 }, "-=.35")
    .from(".hero-art", { autoAlpha: 0, scale: .95, rotate: -1.2, duration: 1.05 }, "-=.85")
    .from(".hero-foot", { autoAlpha: 0, duration: .5 }, "-=.4");

  /* braises */

  const embers = $$(".ember");
  embers.forEach((ember, i) => {
    const rise = gsap.utils.random(46, 78);
    const drift = gsap.utils.random(-12, 12);
    const life = gsap.utils.random(1.7, 2.9);
    gsap.set(ember, { opacity: 0 });
    gsap
      .timeline({ repeat: -1, delay: .8 + i * .42, repeatDelay: gsap.utils.random(.2, 1.1) })
      .fromTo(
        ember,
        { y: 6, x: 0, opacity: 0, scale: .7 },
        { y: -rise * .55, x: drift * .5, opacity: .9, scale: 1, duration: life * .45, ease: "sine.out" }
      )
      .to(ember, { y: -rise, x: drift, opacity: 0, duration: life * .55, ease: "sine.in" });
  });

  /* bandeau */

  const track = $(".marquee-track");
  if (track) {
    gsap.to(track, {
      xPercent: -50,
      duration: 26,
      ease: "none",
      repeat: -1,
    });
  }

  if (!window.ScrollTrigger) return;

  /* labels de section */

  $$(".section-label").forEach((label) => {
    gsap.from(label, {
      x: -14,
      autoAlpha: 0,
      duration: .55,
      ease: "power2.out",
      scrollTrigger: { trigger: label, start: "top 90%", once: true },
    });
  });

  /* blocs révélés */

  $$(".reveal").forEach((el) => {
    gsap.from(el, {
      y: 30,
      autoAlpha: 0,
      duration: .8,
      ease: "power2.out",
      scrollTrigger: { trigger: el, start: "top 87%", once: true },
    });
  });

  /* parallaxe douce des illustrations */

  $$(".feature").forEach((feature) => {
    const art = $(".feature-art", feature);
    if (!art) return;
    gsap.to(art, {
      y: -24,
      ease: "none",
      scrollTrigger: {
        trigger: feature,
        start: "top bottom",
        end: "bottom top",
        scrub: true,
      },
    });
  });

  /* captures */

  const browser = $(".browser-frame");
  if (browser) {
    gsap.from(browser, {
      y: 52,
      autoAlpha: 0,
      rotate: .7,
      duration: .95,
      ease: "power2.out",
      scrollTrigger: { trigger: ".screen-stage", start: "top 80%", once: true },
    });
  }

  $$(".phone").forEach((phone, i) => {
    gsap.from(phone, {
      y: 46,
      autoAlpha: 0,
      duration: .8,
      delay: .12 + i * .12,
      ease: "power2.out",
      scrollTrigger: { trigger: ".screen-stage", start: "top 78%", once: true },
    });
  });

  /* recalcul après chargement des polices et images */

  window.addEventListener("load", () => {
    if (window.ScrollTrigger) ScrollTrigger.refresh();
  });
})();
