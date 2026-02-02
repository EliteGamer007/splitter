"use client"

import React, { createContext, useContext, useEffect, useState } from "react"

type Theme = "dark" | "light"

interface ThemeProviderProps {
    children: React.ReactNode
    defaultTheme?: Theme
    storageKey?: string
}

interface ThemeProviderState {
    theme: Theme
    toggleTheme: () => void
}

const initialState: ThemeProviderState = {
    theme: "dark",
    toggleTheme: () => null,
}

const ThemeProviderContext = createContext<ThemeProviderState>(initialState)

export function ThemeProvider({
    children,
    defaultTheme = "dark",
    storageKey = "theme",
    ...props
}: ThemeProviderProps) {
    const [theme, setTheme] = useState<Theme>(defaultTheme)

    useEffect(() => {
        const savedTheme = (localStorage.getItem(storageKey) as Theme) || defaultTheme
        setTheme(savedTheme)
        document.documentElement.setAttribute("data-theme", savedTheme)
    }, [defaultTheme, storageKey])

    const toggleTheme = () => {
        const nextTheme = theme === "dark" ? "light" : "dark"
        setTheme(nextTheme)
        localStorage.setItem(storageKey, nextTheme)
        document.documentElement.setAttribute("data-theme", nextTheme)
    }

    const value = {
        theme,
        toggleTheme,
    }

    return (
        <ThemeProviderContext.Provider {...props} value={value}>
            {children}
        </ThemeProviderContext.Provider>
    )
}

export const useTheme = () => {
    const context = useContext(ThemeProviderContext)

    if (context === undefined) {
        throw new Error("useTheme must be used within a ThemeProvider")
    }

    return context
}
