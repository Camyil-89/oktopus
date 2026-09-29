import type { ModalProps } from "antd";

/** Должно совпадать с `scrollableModalProps.styles.body.maxHeight`. */
export const MODAL_BODY_MAX_HEIGHT_CSS = "min(70vh, 520px)";

/** Редактор шаблона в модалке: вкладки, подпись поля — скролл только внутри редактора. */
export const MODAL_TEMPLATE_FIELD_HEIGHT_CSS = `calc(${MODAL_BODY_MAX_HEIGHT_CSS} - 9.5rem)`;

export const scrollableModalProps: Pick<
  ModalProps,
  "styles" | "destroyOnClose"
> = {
  destroyOnClose: true,
  styles: {
    body: {
      maxHeight: MODAL_BODY_MAX_HEIGHT_CSS,
      overflowY: "auto",
    },
  },
};
