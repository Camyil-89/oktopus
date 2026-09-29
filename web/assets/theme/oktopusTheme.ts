import type { ThemeConfig } from "antd";
import { theme } from "antd";

const ink = "#121418";
const inkElevated = "#1a1d24";

export const oktopusTheme: ThemeConfig = {
  algorithm: theme.darkAlgorithm,
  token: {
    fontFamily:
      "var(--font-sans), system-ui, -apple-system, sans-serif",
    fontFamilyCode: "var(--font-mono), ui-monospace, monospace",
    colorPrimary: "#5eead4",
    colorBgBase: ink,
    colorBgContainer: "rgba(255,255,255,0.05)",
    colorBgElevated: inkElevated,
    colorBorder: "rgba(255,255,255,0.12)",
    colorBorderSecondary: "rgba(255,255,255,0.08)",
    colorText: "#e4e4e7",
    colorTextSecondary: "#b4b4bb",
    colorTextTertiary: "#94949c",
    colorTextHeading: "#fafafa",
    colorLink: "#5eead4",
    colorLinkHover: "#99f6e4",
    borderRadius: 8,
    borderRadiusLG: 12,
    controlHeight: 36,
    fontSize: 13,
  },
  components: {
    Button: {
      primaryColor: "#0c0d10",
      colorPrimary: "#5eead4",
      colorPrimaryHover: "#99f6e4",
      colorPrimaryActive: "#2dd4bf",
      defaultBg: "transparent",
      defaultBorderColor: "rgba(255,255,255,0.12)",
      defaultColor: "#d4d4d8",
    },
    Card: {
      colorBgContainer: "rgba(255,255,255,0.05)",
      colorBorderSecondary: "rgba(255,255,255,0.08)",
      headerBg: "transparent",
      paddingLG: 20,
    },
    Table: {
      headerBg: "rgba(255,255,255,0.04)",
      headerColor: "#b4b4bb",
      headerSplitColor: "rgba(255,255,255,0.08)",
      rowHoverBg: "rgba(255,255,255,0.04)",
      borderColor: "rgba(255,255,255,0.08)",
      cellPaddingBlock: 12,
      cellPaddingInline: 20,
      colorBgContainer: "transparent",
    },
    Input: {
      colorBgContainer: "rgba(255,255,255,0.05)",
      colorText: "#e4e4e7",
      colorTextPlaceholder: "#71717a",
      activeBorderColor: "rgba(94,234,212,0.45)",
      hoverBorderColor: "rgba(255,255,255,0.18)",
    },
    Select: {
      colorBgContainer: "rgba(255,255,255,0.05)",
      colorText: "#e4e4e7",
      optionSelectedBg: "rgba(94,234,212,0.12)",
    },
    Form: {
      labelColor: "#d4d4d8",
    },
    Modal: {
      contentBg: inkElevated,
      headerBg: inkElevated,
      titleColor: "#fafafa",
    },
    Menu: {
      darkItemBg: "transparent",
      darkItemSelectedBg: "rgba(255,255,255,0.07)",
      darkItemHoverBg: "rgba(255,255,255,0.05)",
      darkItemColor: "#b4b4bb",
      darkItemSelectedColor: "#fafafa",
    },
    Layout: {
      bodyBg: ink,
      headerBg: "rgba(18,20,24,0.85)",
      siderBg: "rgba(18,20,24,0.92)",
    },
    Tag: {
      defaultBg: "rgba(255,255,255,0.06)",
      defaultColor: "#d4d4d8",
    },
    Typography: {
      titleMarginBottom: 0,
      titleMarginTop: 0,
    },
    Descriptions: {
      labelColor: "#b4b4bb",
      contentColor: "#e4e4e7",
    },
    Pagination: {
      colorText: "#d4d4d8",
      colorTextDisabled: "#71717a",
    },
  },
};
