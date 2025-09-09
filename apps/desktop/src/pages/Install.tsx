import { AlertCircle, CheckCircle, XCircle } from 'lucide-react'
import React from 'react'
import { fetchInstallHistory, finalizeInstallation, type InstallInput, type InstallValidation, installCancel, installLogs, installStart, installValidate } from '../api'
import { Alert, AlertDescription, AlertTitle } from '../components/ui/alert'
import { Badge } from '../components/ui/badge'
import { Button } from '../components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Progress } from '../components/ui/progress'
import { ScrollArea } from '../components/ui/scroll-area'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select'

type InstallProgress = {
  id: string
  stage: 'validating' | 'downloading' | 'installing' | 'configuring' | 'finalizing'
  progress: number
  message: string
  logs: string[]
  done: boolean
  success?: boolean
  error?: string
}

type InstallHistory = {
  id: string
  type: string
  uri: string
  slug: string
  timestamp: string
  status: 'completed' | 'failed' | 'cancelled'
  duration: number
}

export function Install() {
  const [source, setSource] = React.useState<InstallInput['type']>('git')
  const [uri, setUri] = React.useState('')
  const [slug, setSlug] = React.useState('')
  const [runtime, setRuntime] = React.useState('auto')
  const [pkgMgr, setPkgMgr] = React.useState('auto')
  const [validation, setValidation] = React.useState<InstallValidation | null>(null)
  const [busy, setBusy] = React.useState(false)
  const [currentJob, setCurrentJob] = React.useState<InstallProgress | null>(null)
  const [history, setHistory] = React.useState<InstallHistory[]>([])
  const [showHistory, setShowHistory] = React.useState(false)
  const [validationInProgress, setValidationInProgress] = React.useState(false)

  const loadHistory = React.useCallback(async () => {
    try {
      const historyData = await fetchInstallHistory()
      setHistory(historyData || [])
    } catch (err) {
      console.warn('Failed to load install history:', err)
      setHistory([]) // Set empty history on error
    }
  }, [])

  React.useEffect(() => {
    loadHistory()
  }, [loadHistory])

  const onValidate = async () => {
    setValidationInProgress(true)
    setValidation(null)
    try {
      const res = await installValidate({ type: source, uri })
      setValidation(res)
      if (res.slug) setSlug(res.slug)
      if (res.runtime) setRuntime(res.runtime)
      if (res.manager) setPkgMgr(res.manager)
    } catch (e) {
      setValidation({ ok: false, problems: [(e as Error).message], slug: '' })
    } finally {
      setValidationInProgress(false)
    }
  }

  const canInstall = validation?.ok && !busy
  const onInstall = async () => {
    if (!canInstall) return
    setBusy(true)
    setCurrentJob({
      id: '',
      stage: 'validating',
      progress: 0,
      message: 'Starting installation...',
      logs: [],
      done: false
    })
    
    try {
      const result = await installStart({ type: source, uri, slug, runtime, manager: pkgMgr })
      const jobId = result?.id || `job-${Date.now()}`
      setCurrentJob(prev => prev ? { ...prev, id: jobId } : null)
      
      const poll = async () => {
        try {
          const res = await installLogs(jobId)
          setCurrentJob(prev => prev ? {
            ...prev,
            logs: res.logs,
            done: res.done,
            success: res.ok,
            error: res.message,
            progress: res.done ? 100 : Math.min(95, (prev.progress || 0) + 5),
            message: res.message || prev.message
          } : null)
          
          if (res.done) {
            setBusy(false)
            if (res.ok) {
              try {
                await finalizeInstallation(jobId)
              } catch (finalizeError) {
                console.warn('Finalization failed, but installation succeeded:', finalizeError)
              }
              await loadHistory() // Refresh history
            }
            setValidation({ ok: res.ok, problems: res.ok ? [] : [res.message || 'Installation failed'], slug })
            return
          }
          setTimeout(poll, 1000)
        } catch (pollError) {
          console.error('Polling error:', pollError)
          setBusy(false)
          setCurrentJob(prev => prev ? {
            ...prev,
            done: true,
            success: false,
            error: (pollError as Error).message
          } : null)
        }
      }
      poll()
    } catch (e) {
      setBusy(false)
      setCurrentJob(prev => prev ? {
        ...prev,
        done: true,
        success: false,
        error: (e as Error).message
      } : null)
      setValidation({ ok: false, problems: [(e as Error).message], slug })
    }
  }

  const onCancel = async () => {
    if (currentJob?.id) {
      try {
        await installCancel(currentJob.id)
        setBusy(false)
        setCurrentJob(prev => prev ? {
          ...prev,
          done: true,
          success: false,
          error: 'Installation cancelled by user'
        } : null)
      } catch (err) {
        console.error('Cancel failed:', err)
      }
    }
  }

  const resetForm = () => {
    setValidation(null)
    setCurrentJob(null)
    setUri('')
    setSlug('')
    setRuntime('')
    setPkgMgr('')
  }

  const getProgressBarColor = (stage: string, success?: boolean) => {
    if (success === false) return 'bg-red-500'
    if (success === true) return 'bg-green-500'
    return 'bg-blue-500'
  }

  const getStageColor = (currentStage: string, stage: string) => {
    if (currentStage === stage) return 'text-blue-600 font-medium'
    const stageOrder = ['validating', 'downloading', 'installing', 'configuring', 'finalizing']
    const currentIndex = stageOrder.indexOf(currentStage)
    const stageIndex = stageOrder.indexOf(stage)
    if (stageIndex < currentIndex) return 'text-green-600'
    return 'text-gray-500'
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Install MCP Server</h1>
          <p className="text-muted-foreground">Install servers from various sources</p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowHistory(!showHistory)}
          >
            {showHistory ? 'Hide History' : 'Show History'}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={resetForm}
            disabled={busy}
          >
            Reset Form
          </Button>
        </div>
      </div>

      {/* Installation History */}
      {showHistory && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base">Installation History</CardTitle>
          </CardHeader>
          <CardContent>
            <ScrollArea className="h-48">
              <div className="space-y-2">
                {history.length > 0 ? (
                  history.map((entry) => (
                    <div key={entry.id} className="flex items-center justify-between p-3 bg-muted/50 rounded-lg text-sm">
                      <div className="flex items-center gap-3">
                        <Badge 
                          variant={
                            entry.status === 'completed' ? 'default' :
                            entry.status === 'failed' ? 'destructive' :
                            'secondary'
                          }
                        >
                          {entry.status}
                        </Badge>
                        <span className="font-medium">{entry.slug}</span>
                        <span className="text-muted-foreground">({entry.type})</span>
                      </div>
                      <div className="text-muted-foreground text-xs">
                        {new Date(entry.timestamp).toLocaleDateString()}
                      </div>
                    </div>
                  ))
                ) : (
                  <div className="text-center py-8 text-muted-foreground text-sm">
                    No installation history available
                  </div>
                )}
              </div>
            </ScrollArea>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Installation Form */}
        <Card>
          <CardHeader>
            <CardTitle>Configuration</CardTitle>
            <CardDescription>Configure the MCP server installation</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="source-type">Source Type</Label>
              <Select value={source} onValueChange={(value) => setSource(value as any)} disabled={busy}>
                <SelectTrigger>
                  <SelectValue placeholder="Select source type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="git">Git Repository</SelectItem>
                  <SelectItem value="npm">NPM Package</SelectItem>
                  <SelectItem value="pip">Python Package (pip)</SelectItem>
                  <SelectItem value="docker-image">Docker Image</SelectItem>
                  <SelectItem value="docker-compose">Docker Compose</SelectItem>
                </SelectContent>
              </Select>
            </div>
            
            <div className="space-y-2">
              <Label htmlFor="uri">URI / Path</Label>
              <Input 
                id="uri"
                placeholder={
                  source === 'git' ? 'https://github.com/user/repo' :
                  source === 'npm' ? 'package-name' :
                  source === 'pip' ? 'package-name' :
                  source === 'docker-image' ? 'ghcr.io/user/image:tag' :
                  '/path/to/compose.yml'
                }
                value={uri} 
                onChange={(e) => setUri(e.target.value)}
                disabled={busy}
              />
            </div>
            
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="slug">Slug (ID)</Label>
                <Input 
                  id="slug"
                  placeholder="server-name" 
                  value={slug} 
                  onChange={(e) => setSlug(e.target.value)}
                  disabled={busy}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="runtime">Runtime</Label>
                <Select value={runtime} onValueChange={setRuntime} disabled={busy}>
                  <SelectTrigger>
                    <SelectValue placeholder="Auto-detect" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="auto">Auto-detect</SelectItem>
                    <SelectItem value="node">Node.js</SelectItem>
                    <SelectItem value="python">Python</SelectItem>
                    <SelectItem value="docker">Docker</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            
            <div className="space-y-2">
              <Label htmlFor="package-manager">Package Manager</Label>
              <Select value={pkgMgr} onValueChange={setPkgMgr} disabled={busy}>
                <SelectTrigger>
                  <SelectValue placeholder="Auto-detect" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="auto">Auto-detect</SelectItem>
                  <SelectItem value="npm">npm</SelectItem>
                  <SelectItem value="pnpm">pnpm</SelectItem>
                  <SelectItem value="yarn">yarn</SelectItem>
                  <SelectItem value="pip">pip</SelectItem>
                  <SelectItem value="uv">uv</SelectItem>
                  <SelectItem value="pipx">pipx</SelectItem>
                </SelectContent>
              </Select>
            </div>
            
            <div className="flex gap-3 pt-2">
              <Button 
                variant="outline"
                onClick={onValidate} 
                disabled={busy || !uri.trim() || validationInProgress}
              >
                {validationInProgress ? 'Validating...' : 'Validate'}
              </Button>
              <Button 
                onClick={onInstall} 
                disabled={!canInstall}
              >
                Install
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Validation Results */}
        {validation && (
          <Alert variant={validation.ok ? "default" : "destructive"}>
            {validation.ok ? <CheckCircle className="h-4 w-4" /> : <XCircle className="h-4 w-4" />}
            <AlertTitle>
              {validation.ok ? 'Validation Successful' : 'Validation Failed'}
            </AlertTitle>
            {validation.problems && validation.problems.length > 0 && (
              <AlertDescription>
                <ul className="mt-2 space-y-1 list-disc list-inside">
                  {validation.problems.map((problem, i) => (
                    <li key={i}>{problem}</li>
                  ))}
                </ul>
              </AlertDescription>
            )}
          </Alert>
        )}
        </div>

        {/* Progress and Logs */}
        <Card>
          <CardHeader>
            <CardTitle>Installation Progress</CardTitle>
            <CardDescription>Monitor the installation process</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
          
          {currentJob && (
            <div className="space-y-4">
              {/* Progress Bar */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium">Progress</span>
                  <span className="text-sm text-muted-foreground">{currentJob.progress}%</span>
                </div>
                <Progress 
                  value={currentJob.progress} 
                  className="h-2"
                />
              </div>

              {/* Stage Indicators */}
              <div className="flex justify-between text-xs">
                {['validating', 'downloading', 'installing', 'configuring', 'finalizing'].map((stage) => (
                  <div key={stage} className={`text-center ${getStageColor(currentJob.stage, stage)}`}>
                    <div className="capitalize">{stage}</div>
                  </div>
                ))}
              </div>

              {/* Current Status */}
              <div className="bg-muted p-3 rounded-lg">
                <div className="text-sm">
                  <span className="font-medium">Status:</span> {currentJob.message}
                </div>
                {currentJob.error && (
                  <div className="text-sm text-destructive mt-1">
                    <span className="font-medium">Error:</span> {currentJob.error}
                  </div>
                )}
              </div>

              {/* Action Buttons */}
              {!currentJob.done && (
                <div className="flex gap-2">
                  <Button
                    variant="destructive"
                    onClick={onCancel}
                  >
                    Cancel Installation
                  </Button>
                </div>
              )}
            </div>
          )}

          {/* Installation Logs */}
          <div>
            <div className="text-sm font-medium mb-2">Logs</div>
            <ScrollArea className="h-80 w-full border rounded-lg">
              <div className="p-3 text-xs bg-black text-green-400 font-mono min-h-full">
                {currentJob?.logs.length ? (
                  currentJob.logs.map((line, i) => (
                    <div key={i} className="mb-1 break-all">{line}</div>
                  ))
                ) : validation ? (
                  <pre className="text-gray-300 whitespace-pre-wrap">{JSON.stringify(validation, null, 2)}</pre>
                ) : (
                  <div className="text-gray-500">
                    {busy ? 'Starting installation...' : 'Ready to install. Click Validate to begin.'}
                  </div>
                )}
              </div>
            </ScrollArea>
          </div>
          </CardContent>
        </Card>
      </div>
  )
}
