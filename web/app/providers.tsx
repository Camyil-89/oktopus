"use client";

import { AntdRegistry } from "@ant-design/nextjs-registry";
import { App, ConfigProvider } from "antd";
import { OktopusAntdIndicator } from "@/assets/components/oktopus/OktopusAntdIndicator";
import { oktopusTheme } from "@/assets/theme/oktopusTheme";
import { AuthProvider } from "@/contexts/AuthContext";

const oktopusIndicator = <OktopusAntdIndicator />;

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <AntdRegistry>
      <ConfigProvider
        theme={oktopusTheme}
        spin={{ indicator: oktopusIndicator }}
        button={{ loadingIcon: oktopusIndicator }}
        select={{ loadingIcon: oktopusIndicator }}
      >
        <App className="flex h-full min-h-full flex-1 flex-col">
          <AuthProvider>
            <div className="flex h-full min-h-full flex-1 flex-col">
              {children}
            </div>
          </AuthProvider>
        </App>
      </ConfigProvider>
    </AntdRegistry>
  );
}
