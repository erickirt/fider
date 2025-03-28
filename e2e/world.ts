import { World as CucumberWorld } from "@cucumber/cucumber"
import { Page } from "@playwright/test"

export interface FiderWorld extends CucumberWorld {
  tenantName: string
  page: Page
  log: (msg: string) => void
}
