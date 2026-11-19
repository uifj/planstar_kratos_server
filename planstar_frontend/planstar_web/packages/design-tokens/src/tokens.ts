/**
 * Planstar 设计令牌 —— 与 Flutter 端 `core_design_system` 一一对应。
 *
 * 三层结构：
 * 1. `palette`：L0 原始色板，只有这一层写死十六进制；
 * 2. `semanticColors`：L1 语义槽，组件只准引用这一层；
 * 3. `typography` / `spacing` / `radius`：字号、间距、圆角阶梯。
 *
 * Web 端样式实际消费的是 `theme.css` 中的 CSS 变量；
 * 本文件供需要在 JS 中取色的场景（图表、meta theme-color、内联样式）使用。
 */

export const palette = {
  black1: '#000000',
  black2: 'rgba(0, 0, 0, 0.90)',
  black3: 'rgba(0, 0, 0, 0.72)',
  black4: 'rgba(0, 0, 0, 0.55)',
  black5: 'rgba(0, 0, 0, 0.40)',
  black6: 'rgba(0, 0, 0, 0.25)',
  black7: 'rgba(0, 0, 0, 0.12)',
  black8: 'rgba(0, 0, 0, 0.06)',

  white1: '#FFFFFF',
  white2: 'rgba(255, 255, 255, 0.93)',
  white3: 'rgba(255, 255, 255, 0.75)',
  white4: 'rgba(255, 255, 255, 0.55)',
  white5: 'rgba(255, 255, 255, 0.42)',
  white6: 'rgba(255, 255, 255, 0.30)',
  white7: 'rgba(255, 255, 255, 0.14)',

  grayLight1: '#F9FAFC',
  grayLight2: '#F0F2F7',
  grayLight3: '#E6E9F0',
  grayLight4: '#D1D4DE',
  grayLight5: '#C0C3CC',
  grayLight6: '#B3B6BE',
  grayLight7: '#A5A9B0',

  grayDark1: '#131417',
  grayDark2: '#1F2024',
  grayDark3: '#2B2C30',
  grayDark4: '#3A3C42',
  grayDark5: '#48494F',
  grayDark6: '#54565C',
  grayDark7: '#646A70',

  themeLight1: '#EBF3FF',
  themeLight2: '#CCE2FF',
  themeLight3: '#ADCFFF',
  themeLight4: '#7AAFFF',
  themeLight5: '#4588F5',
  themeLight6: '#1C66E5',
  themeLight7: '#0D49BF',
  themeLight8: '#033099',

  themeDark1: '#1C2333',
  themeDark2: '#243047',
  themeDark3: '#2F4875',
  themeDark4: '#305BA6',
  themeDark5: '#2B6AD6',
  themeDark6: '#4086FF',
  themeDark7: '#5C9DFF',

  greenLight1: '#E1F7EB',
  greenLight6: '#0ABF77',
  greenDark2: '#22352C',
  greenDark6: '#38A673',

  redLight1: '#FFE7E6',
  redLight6: '#E54545',
  redDark6: '#E6594C',

  orangeLight1: '#FFEEDB',
  orangeLight6: '#FF7200',
  orangeDark2: '#35231A',
  orangeDark6: '#E37F32',
} as const;

/** L1 语义色槽，与 Flutter `AppColors` 的字段完全同名。 */
export interface SemanticColors {
  textPrimary: string;
  textSecondary: string;
  textTertiary: string;
  textDisabled: string;
  bgPage: string;
  bgCard: string;
  bgSubtle: string;
  bgOverlay: string;
  divider: string;
  border: string;
  brand: string;
  brandPressed: string;
  brandDisabled: string;
  brandSubtle: string;
  onBrand: string;
  price: string;
  success: string;
  warning: string;
  warningBg: string;
  error: string;
  info: string;
  skeleton: string;
  tabActive: string;
  tabInactive: string;
}

export const lightColors: SemanticColors = {
  textPrimary: palette.black2,
  textSecondary: palette.black4,
  textTertiary: palette.black5,
  textDisabled: palette.black6,
  bgPage: palette.grayLight1,
  bgCard: palette.white1,
  bgSubtle: palette.grayLight2,
  bgOverlay: palette.black4,
  divider: palette.grayLight2,
  border: palette.grayLight3,
  brand: palette.themeLight6,
  brandPressed: palette.themeLight7,
  brandDisabled: palette.themeLight2,
  brandSubtle: palette.themeLight1,
  onBrand: palette.white1,
  price: palette.redLight6,
  success: palette.greenLight6,
  warning: palette.orangeLight6,
  warningBg: palette.orangeLight1,
  error: palette.redLight6,
  info: palette.themeLight5,
  skeleton: palette.grayLight2,
  tabActive: palette.themeLight6,
  tabInactive: palette.black4,
};

export const darkColors: SemanticColors = {
  textPrimary: palette.white2,
  textSecondary: palette.white4,
  textTertiary: palette.white6,
  textDisabled: palette.white7,
  bgPage: palette.grayDark1,
  bgCard: palette.grayDark2,
  bgSubtle: palette.grayDark3,
  bgOverlay: palette.black4,
  divider: palette.grayDark3,
  border: palette.grayDark4,
  brand: palette.themeDark6,
  brandPressed: palette.themeDark7,
  brandDisabled: palette.themeDark2,
  brandSubtle: palette.themeDark2,
  onBrand: palette.white1,
  price: palette.redDark6,
  success: palette.greenDark6,
  warning: palette.orangeDark6,
  warningBg: palette.orangeDark2,
  error: palette.redDark6,
  info: palette.themeDark6,
  skeleton: palette.grayDark3,
  tabActive: palette.themeDark6,
  tabInactive: palette.white4,
};

/** 字号阶梯（px）。 */
export const fontSize = {
  display: 32,
  title: 20,
  subtitle: 16,
  body: 14,
  caption: 12,
  label: 10,
} as const;

/** 字重（只保留三档）。 */
export const fontWeight = {
  regular: 400,
  medium: 500,
  bold: 700,
} as const;

/** 间距阶梯（px，4 基准）。 */
export const spacing = {
  xxs: 4,
  xs: 8,
  sm: 12,
  md: 16,
  lg: 20,
  xl: 24,
  xxl: 32,
  section: 40,
  huge: 56,
  giant: 72,
} as const;

/** 圆角阶梯（px）。 */
export const radius = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 20,
  full: 999,
} as const;

/** 字体族：品牌字体缺失时按中文系统字体逐级降级。 */
export const fontFamily =
  '"ChillRoundGothic", "PingFang SC", "Noto Sans SC", "Microsoft YaHei", system-ui, sans-serif';

/** 官网内容区最大宽度。 */
export const layout = {
  maxWidth: 1200,
  headerHeight: 64,
  mobilePadding: 16,
} as const;
