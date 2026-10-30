export const lessonTheme = `<style id="repet-unified-theme">
:root{--bg:#f5f7f7;--surface:#ffffff;--ink:#27322d;--text:#27322d;--muted:#66736c;--green:#3f7659;--green-hover:#315f49;--primary:#3f7659;--selected:#edf2eb;--border:#dfe6e2;--orange:#b94f20;--wrong:#b3443e;--shadow:none}
html{background:#f5f7f7}body{background:#f5f7f7;color:#27322d;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;font-size:15px;line-height:1.6}
body,body *{box-sizing:border-box;letter-spacing:0}
body :is(h1,h2,h3,h4,p,label,legend,dd){overflow-wrap:anywhere}
body :is(h1,.h1,.heading){font-size:30px;line-height:1.2;overflow-wrap:anywhere}
body :is(h2,.section-heading){font-size:22px;line-height:1.3}
body :is(button,input,select,textarea){font:inherit;min-width:0;max-width:100%}body button{min-height:44px;white-space:normal;overflow-wrap:anywhere;line-height:1.4}
body :is(button,a,input,select,textarea):focus-visible{outline:3px solid #89b298;outline-offset:3px}
body .side{background:#ffffff;border-right:1px solid #dfe6e2;box-shadow:none}
body .nav-btn.active{background:#edf2eb;color:#315f49;border-color:#cadac8;box-shadow:none}
body .nav-btn.active::before{background:#3f7659}
body :is(.experience-shell,.text-block,.quiz,.card,.panel){min-width:0;box-shadow:none;border-color:#dfe6e2;border-radius:8px}
body .hero{background:#edf2eb;border-color:#dce6d9;box-shadow:none;border-radius:8px}
body .hero::after{display:none}body .repet-logo{border:0;padding:0;background:transparent;border-radius:0}
body :is(.experience-head,.topbar,.quiz-head){flex-wrap:wrap}
body :is(.main,.page,.experience-lab,.experience-note,.experience-columns>*){min-width:0}
@media(max-width:700px){body :is(h1,.h1,.heading){font-size:26px}body .app{display:block}body .side{height:auto;position:relative;width:auto;border-right:0;border-bottom:1px solid #dfe6e2}body .main{padding:20px 14px 40px}body .experience-columns{grid-template-columns:1fr}body .experience-shell{padding:18px}body .nav-btn{min-height:44px}}
</style>`;
