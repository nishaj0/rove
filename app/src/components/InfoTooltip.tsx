import { Info } from "lucide-react";
export function InfoTooltip({ label }: { label: string }) { return <span className="info-tooltip"><button type="button" className="info-tooltip-button" aria-label={label}><Info aria-hidden="true" size={14} /></button><span role="tooltip">{label}</span></span>; }
