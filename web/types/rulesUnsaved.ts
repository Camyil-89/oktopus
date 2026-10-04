export type RulesSectionUnsaved = {
  dirty: boolean;
  discard: () => void;
  save: () => Promise<boolean>;
};
