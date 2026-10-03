"use client";

import { AntdRegistry } from "@ant-design/nextjs-registry";
import { App, ConfigProvider } from "antd";
import enUS from "antd/locale/en_US";
import ruRU from "antd/locale/ru_RU";
import { OktopusAntdIndicator } from "@/assets/components/oktopus/OktopusAntdIndicator";
import { oktopusTheme } from "@/assets/theme/oktopusTheme";
import { AuthProvider } from "@/contexts/AuthContext";
import { LocaleProvider, useLocale } from "@/contexts/LocaleContext";

const oktopusIndicator = <OktopusAntdIndicator />;

function AntdLocaleBridge({ children }: { children: React.ReactNode }) {
  const { locale } = useLocale();
  const antdLocale = locale === "en" ? enUS : ruRU;

  return (
    <ConfigProvider
      locale={antdLocale}
      theme={oktopusTheme}
      spin={{ indicator: oktopusIndicator }}
      button={{ loadingIcon: oktopusIndicator }}
      select={{ loadingIcon: oktopusIndicator }}
    >
      <App className="flex h-full min-h-full flex-1 flex-col">{children}</App>
    </ConfigProvider>
  );
}

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <AntdRegistry>
      <LocaleProvider>
        <AntdLocaleBridge>
          <AuthProvider>
            <div className="flex h-full min-h-full flex-1 flex-col">
              {children}
            </div>
          </AuthProvider>
        </AntdLocaleBridge>
      </LocaleProvider>
    </AntdRegistry>
  );
}
