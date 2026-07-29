import * as React from "react"
import { createPortal } from "react-dom"

import type { ThemeComponent, ThemeRegistry } from "@thism/theme-sdk"
import { Badge } from "../components/ui/badge"
import { Button, type ButtonProps } from "../components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "../components/ui/card"
import {
  Dialog as DialogRoot,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogOverlay,
  DialogPortal,
  DialogTitle,
  DialogTrigger,
} from "../components/ui/dialog"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectSeparator, SelectTrigger, SelectValue } from "../components/ui/select"
import { Table, TableBody, TableCaption, TableCell, TableFooter, TableHead, TableHeader, TableRow } from "../components/ui/table"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs"
import type { ThemePluginModule } from "./types"

type DefaultButtonProps = ButtonProps & { loading?: boolean }

const DefaultButton = React.forwardRef<HTMLButtonElement, DefaultButtonProps>(function DefaultButton(
  { children, disabled, loading, variant, ...props },
  ref,
) {
  return (
    <Button
      {...props}
      ref={ref}
      variant={variant}
      data-variant={variant}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
    >
      {children}
    </Button>
  )
}) as ThemeComponent

const DefaultInput = Input as ThemeComponent

type DefaultCheckboxProps = React.InputHTMLAttributes<HTMLInputElement> & { children?: React.ReactNode }

const DefaultCheckbox = React.forwardRef<HTMLInputElement, DefaultCheckboxProps>(function DefaultCheckbox(
  { children, className, ...props },
  ref,
) {
  return (
    <label className={className}>
      <input {...props} ref={ref} type="checkbox" />
      {children}
    </label>
  )
}) as ThemeComponent

type DefaultSwitchProps = React.ButtonHTMLAttributes<HTMLButtonElement> & { checked?: boolean }

const DefaultSwitch = React.forwardRef<HTMLButtonElement, DefaultSwitchProps>(function DefaultSwitch(
  { children, checked, ...props },
  ref,
) {
  return (
    <button {...props} ref={ref} type="button" role="switch" aria-checked={Boolean(checked)}>
      {children}
    </button>
  )
}) as ThemeComponent

type DefaultDialogProps = {
  children?: React.ReactNode
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

function DefaultDialog({ children, open, onOpenChange }: DefaultDialogProps) {
  return (
    <DialogRoot open={Boolean(open)} onOpenChange={onOpenChange}>
      <DialogContent aria-modal="true">
        <DialogTitle className="sr-only">Theme dialog</DialogTitle>
        <DialogDescription className="sr-only">Theme-provided dialog surface</DialogDescription>
        {children}
      </DialogContent>
    </DialogRoot>
  )
}

const DefaultCardRoot = React.forwardRef<React.ElementRef<typeof Card>, React.ComponentPropsWithoutRef<typeof Card>>(function DefaultCardRoot(props, ref) {
  return <Card {...props} ref={ref} />
})

const DefaultCard = Object.assign(DefaultCardRoot, {
  Header: CardHeader,
  Title: CardTitle,
  Description: CardDescription,
  Content: CardContent,
  Footer: CardFooter,
}) as ThemeComponent

function DefaultSelectRoot(props: React.ComponentProps<typeof Select>) {
  return <Select {...props} />
}

const DefaultSelect = Object.assign(DefaultSelectRoot, {
  Trigger: SelectTrigger,
  Value: SelectValue,
  Content: SelectContent,
  Group: SelectGroup,
  Label: SelectLabel,
  Item: SelectItem,
  Separator: SelectSeparator,
}) as ThemeComponent

function DefaultTabsRoot(props: React.ComponentProps<typeof Tabs>) {
  return <Tabs {...props} />
}

const DefaultTabs = Object.assign(DefaultTabsRoot, { List: TabsList, Trigger: TabsTrigger, Content: TabsContent }) as ThemeComponent

const DefaultTableRoot = React.forwardRef<React.ElementRef<typeof Table>, React.ComponentPropsWithoutRef<typeof Table>>(function DefaultTableRoot(props, ref) {
  return <Table {...props} ref={ref} />
})

const DefaultTable = Object.assign(DefaultTableRoot, {
  Header: TableHeader,
  Body: TableBody,
  Footer: TableFooter,
  Head: TableHead,
  Row: TableRow,
  Cell: TableCell,
  Caption: TableCaption,
}) as ThemeComponent

const DefaultCompoundDialog = Object.assign(DefaultDialog, {
  Trigger: DialogTrigger,
  Portal: DialogPortal,
  Overlay: DialogOverlay,
  Content: DialogContent,
  Header: DialogHeader,
  Footer: DialogFooter,
  Title: DialogTitle,
  Description: DialogDescription,
  Close: DialogClose,
}) as ThemeComponent

const DefaultForm = React.forwardRef<HTMLFormElement, React.FormHTMLAttributes<HTMLFormElement>>(function DefaultForm(props, ref) {
  return <form {...props} ref={ref} />
}) as ThemeComponent

const DefaultSurface = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(function DefaultSurface(props, ref) {
  return <div {...props} ref={ref} />
}) as ThemeComponent

const DefaultDropdownMenu = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(function DefaultDropdownMenu(props, ref) {
  return <div {...props} ref={ref} role="menu" />
}) as ThemeComponent

const DefaultTooltip = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(function DefaultTooltip(props, ref) {
  return <div {...props} ref={ref} role="tooltip" />
}) as ThemeComponent

const DefaultPopover = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(function DefaultPopover(props, ref) {
  return <div {...props} ref={ref} role="dialog" data-state="open" />
}) as ThemeComponent

const DefaultScrollArea = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(function DefaultScrollArea({ style, ...props }, ref) {
  return <div {...props} ref={ref} style={{ overflow: "auto", ...style }} />
}) as ThemeComponent

const DefaultToast = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(function DefaultToast(props, ref) {
  return <div {...props} ref={ref} role="status" aria-live="polite" />
}) as ThemeComponent

const DefaultSkeleton = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(function DefaultSkeleton(props, ref) {
  return <div {...props} ref={ref} aria-hidden="true" />
}) as ThemeComponent

const DefaultSeparator = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(function DefaultSeparator(props, ref) {
  return <div {...props} ref={ref} role="separator" />
}) as ThemeComponent

const DefaultResponsiveNavigation = React.forwardRef<HTMLElement, React.HTMLAttributes<HTMLElement>>(function DefaultResponsiveNavigation(props, ref) {
  return <nav {...props} ref={ref} data-responsive="true" />
}) as ThemeComponent

const DefaultRootShell = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(function DefaultRootShell(props, ref) {
  return <div {...props} ref={ref} data-thism-shell="root" />
}) as ThemeComponent

const DefaultPrimaryNavigation = React.forwardRef<HTMLElement, React.HTMLAttributes<HTMLElement>>(function DefaultPrimaryNavigation(props, ref) {
  return <nav {...props} ref={ref} data-thism-navigation="primary" />
}) as ThemeComponent

const DefaultMobileNavigation = React.forwardRef<HTMLElement, React.HTMLAttributes<HTMLElement>>(function DefaultMobileNavigation(props, ref) {
  return <nav {...props} ref={ref} data-thism-navigation="mobile" />
}) as ThemeComponent

const DefaultPageHeader = React.forwardRef<HTMLElement, React.HTMLAttributes<HTMLElement>>(function DefaultPageHeader(props, ref) {
  return <header {...props} ref={ref} />
}) as ThemeComponent

const DefaultSettingsSection = React.forwardRef<HTMLElement, React.HTMLAttributes<HTMLElement>>(function DefaultSettingsSection(props, ref) {
  return <section {...props} ref={ref} />
}) as ThemeComponent

const DefaultStatusState = React.forwardRef<HTMLElement, React.HTMLAttributes<HTMLElement>>(function DefaultStatusState(props, ref) {
  return <section {...props} ref={ref} role="status" />
}) as ThemeComponent

const DefaultErrorState = React.forwardRef<HTMLElement, React.HTMLAttributes<HTMLElement>>(function DefaultErrorState(props, ref) {
  return <section {...props} ref={ref} role="alert" />
}) as ThemeComponent

function portalComponent(surface: "dialog" | "sheet" | "menu" | "toast") {
  return function DefaultPortal({ children }: { children?: React.ReactNode }) {
    if (typeof document === "undefined") return null
    return createPortal(<div data-thism-theme-portal={surface}>{children}</div>, document.body)
  }
}

const defaultShadcnRegistry: ThemeRegistry = {
  apiVersion: "1.0.0",
  primitives: {
    Button: DefaultButton,
    Badge: Badge as ThemeComponent,
    Card: DefaultCard,
    Input: DefaultInput,
    Select: DefaultSelect,
    Checkbox: DefaultCheckbox,
    Switch: DefaultSwitch,
    Tabs: DefaultTabs,
    Dialog: DefaultCompoundDialog,
    Sheet: DefaultCompoundDialog,
    DropdownMenu: DefaultDropdownMenu,
    Tooltip: DefaultTooltip,
    Popover: DefaultPopover,
    Table: DefaultTable,
    Form: DefaultForm,
    Toast: DefaultToast,
    Skeleton: DefaultSkeleton,
    Separator: DefaultSeparator,
    ScrollArea: DefaultScrollArea,
    ResponsiveNavigation: DefaultResponsiveNavigation,
  },
  shells: {
    RootShell: DefaultRootShell,
    PrimaryNavigation: DefaultPrimaryNavigation,
    MobileNavigation: DefaultMobileNavigation,
    PageContainer: DefaultSurface,
    PageHeader: DefaultPageHeader,
    SettingsSection: DefaultSettingsSection,
    DashboardGrid: DefaultSurface,
    EmptyState: DefaultStatusState,
    LoadingState: DefaultStatusState,
    ErrorState: DefaultErrorState,
  },
  slots: {
    Brand: DefaultSurface,
    UserMenu: DefaultSurface,
    ThemeControls: DefaultSurface,
    NotificationCenter: DefaultSurface,
  },
  portals: {
    DialogPortal: portalComponent("dialog") as ThemeComponent,
    SheetPortal: portalComponent("sheet") as ThemeComponent,
    MenuPortal: portalComponent("menu") as ThemeComponent,
    ToastPortal: portalComponent("toast") as ThemeComponent,
  },
  tokens: {
    light: {
      background: "0 0% 100%",
      foreground: "222 47% 11%",
      card: "0 0% 100%",
      border: "214 32% 91%",
      primary: "221 83% 53%",
    },
    dark: {
      background: "222 47% 11%",
      foreground: "210 40% 98%",
      card: "222 47% 11%",
      border: "217 33% 18%",
      primary: "217 91% 60%",
    },
  },
  settings: [],
}

export const DEFAULT_SHADCN_PLUGIN: ThemePluginModule = Object.freeze({
  id: "default-shadcn",
  name: "Default shadcn",
  version: "1.0.0",
  source: "embedded",
  removable: false,
  activeByDefault: true,
  registry: defaultShadcnRegistry,
})

export const EMBEDDED_THEME_PLUGINS: readonly ThemePluginModule[] = Object.freeze([DEFAULT_SHADCN_PLUGIN])
