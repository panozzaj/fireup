// morphdom v2.7.4 - https://github.com/patrick-steele-idem/morphdom
;(function (global, factory) {
    typeof exports === 'object' && typeof module !== 'undefined'
        ? (module.exports = factory())
        : typeof define === 'function' && define.amd
          ? define(factory)
          : ((global = global || self), (global.morphdom = factory()))
})(this, function () {
    'use strict'
    var DOCUMENT_FRAGMENT_NODE = 11
    function morphAttrs(fromNode, toNode) {
        var toNodeAttrs = toNode.attributes
        var attr
        var attrName
        var attrNamespaceURI
        var attrValue
        var fromValue
        if (toNode.nodeType === DOCUMENT_FRAGMENT_NODE || fromNode.nodeType === DOCUMENT_FRAGMENT_NODE) {
            return
        }
        for (var i = toNodeAttrs.length - 1; i >= 0; i--) {
            attr = toNodeAttrs[i]
            attrName = attr.name
            attrNamespaceURI = attr.namespaceURI
            attrValue = attr.value
            if (attrNamespaceURI) {
                attrName = attr.localName || attrName
                fromValue = fromNode.getAttributeNS(attrNamespaceURI, attrName)
                if (fromValue !== attrValue) {
                    if (attr.prefix === 'xmlns') {
                        attrName = attr.name
                    }
                    fromNode.setAttributeNS(attrNamespaceURI, attrName, attrValue)
                }
            } else {
                fromValue = fromNode.getAttribute(attrName)
                if (fromValue !== attrValue) {
                    fromNode.setAttribute(attrName, attrValue)
                }
            }
        }
        var fromNodeAttrs = fromNode.attributes
        for (var d = fromNodeAttrs.length - 1; d >= 0; d--) {
            attr = fromNodeAttrs[d]
            attrName = attr.name
            attrNamespaceURI = attr.namespaceURI
            if (attrNamespaceURI) {
                attrName = attr.localName || attrName
                if (!toNode.hasAttributeNS(attrNamespaceURI, attrName)) {
                    fromNode.removeAttributeNS(attrNamespaceURI, attrName)
                }
            } else {
                if (!toNode.hasAttribute(attrName)) {
                    fromNode.removeAttribute(attrName)
                }
            }
        }
    }
    var range
    var NS_XHTML = 'http://www.w3.org/1999/xhtml'
    var doc = typeof document === 'undefined' ? undefined : document
    var HAS_TEMPLATE_SUPPORT = !!doc && 'content' in doc.createElement('template')
    var HAS_RANGE_SUPPORT = !!doc && doc.createRange && 'createContextualFragment' in doc.createRange()
    function createFragmentFromTemplate(str) {
        var template = doc.createElement('template')
        template.innerHTML = str
        return template.content.childNodes[0]
    }
    function createFragmentFromRange(str) {
        if (!range) {
            range = doc.createRange()
            range.selectNode(doc.body)
        }
        var fragment = range.createContextualFragment(str)
        return fragment.childNodes[0]
    }
    function createFragmentFromWrap(str) {
        var fragment = doc.createElement('body')
        fragment.innerHTML = str
        return fragment.childNodes[0]
    }
    function toElement(str) {
        str = str.trim()
        if (HAS_TEMPLATE_SUPPORT) {
            return createFragmentFromTemplate(str)
        } else if (HAS_RANGE_SUPPORT) {
            return createFragmentFromRange(str)
        }
        return createFragmentFromWrap(str)
    }
    function compareNodeNames(fromEl, toEl) {
        var fromNodeName = fromEl.nodeName
        var toNodeName = toEl.nodeName
        var fromCodeStart, toCodeStart
        if (fromNodeName === toNodeName) {
            return true
        }
        fromCodeStart = fromNodeName.charCodeAt(0)
        toCodeStart = toNodeName.charCodeAt(0)
        if (fromCodeStart <= 90 && toCodeStart >= 97) {
            return fromNodeName === toNodeName.toUpperCase()
        } else if (toCodeStart <= 90 && fromCodeStart >= 97) {
            return toNodeName === fromNodeName.toUpperCase()
        } else {
            return false
        }
    }
    function createElementNS(name, namespaceURI) {
        return !namespaceURI || namespaceURI === NS_XHTML
            ? doc.createElement(name)
            : doc.createElementNS(namespaceURI, name)
    }
    function moveChildren(fromEl, toEl) {
        var curChild = fromEl.firstChild
        while (curChild) {
            var nextChild = curChild.nextSibling
            toEl.appendChild(curChild)
            curChild = nextChild
        }
        return toEl
    }
    function syncBooleanAttrProp(fromEl, toEl, name) {
        if (fromEl[name] !== toEl[name]) {
            fromEl[name] = toEl[name]
            if (fromEl[name]) {
                fromEl.setAttribute(name, '')
            } else {
                fromEl.removeAttribute(name)
            }
        }
    }
    var specialElHandlers = {
        OPTION: function (fromEl, toEl) {
            var parentNode = fromEl.parentNode
            if (parentNode) {
                var parentName = parentNode.nodeName.toUpperCase()
                if (parentName === 'OPTGROUP') {
                    parentNode = parentNode.parentNode
                    parentName = parentNode && parentNode.nodeName.toUpperCase()
                }
                if (parentName === 'SELECT' && !parentNode.hasAttribute('multiple')) {
                    if (fromEl.hasAttribute('selected') && !toEl.selected) {
                        fromEl.setAttribute('selected', 'selected')
                        fromEl.removeAttribute('selected')
                    }
                    parentNode.selectedIndex = -1
                }
            }
            syncBooleanAttrProp(fromEl, toEl, 'selected')
        },
        INPUT: function (fromEl, toEl) {
            syncBooleanAttrProp(fromEl, toEl, 'checked')
            syncBooleanAttrProp(fromEl, toEl, 'disabled')
            if (fromEl.value !== toEl.value) {
                fromEl.value = toEl.value
            }
            if (!toEl.hasAttribute('value')) {
                fromEl.removeAttribute('value')
            }
        },
        TEXTAREA: function (fromEl, toEl) {
            var newValue = toEl.value
            if (fromEl.value !== newValue) {
                fromEl.value = newValue
            }
            var firstChild = fromEl.firstChild
            if (firstChild) {
                var oldValue = firstChild.nodeValue
                if (oldValue == newValue || (!newValue && oldValue == fromEl.placeholder)) {
                    return
                }
                firstChild.nodeValue = newValue
            }
        },
        SELECT: function (fromEl, toEl) {
            if (!toEl.hasAttribute('multiple')) {
                var selectedIndex = -1
                var i = 0
                var curChild = fromEl.firstChild
                var optgroup
                var nodeName
                while (curChild) {
                    nodeName = curChild.nodeName && curChild.nodeName.toUpperCase()
                    if (nodeName === 'OPTGROUP') {
                        optgroup = curChild
                        curChild = optgroup.firstChild
                    } else {
                        if (nodeName === 'OPTION') {
                            if (curChild.hasAttribute('selected')) {
                                selectedIndex = i
                                break
                            }
                            i++
                        }
                        curChild = curChild.nextSibling
                        if (!curChild && optgroup) {
                            curChild = optgroup.nextSibling
                            optgroup = null
                        }
                    }
                }
                fromEl.selectedIndex = selectedIndex
            }
        },
    }
    var ELEMENT_NODE = 1
    var DOCUMENT_FRAGMENT_NODE$1 = 11
    var TEXT_NODE = 3
    var COMMENT_NODE = 8
    function noop() {}
    function defaultGetNodeKey(node) {
        if (node) {
            return (node.getAttribute && node.getAttribute('id')) || node.id
        }
    }
    function morphdomFactory(morphAttrs) {
        return function morphdom(fromNode, toNode, options) {
            if (!options) {
                options = {}
            }
            if (typeof toNode === 'string') {
                if (fromNode.nodeName === '#document' || fromNode.nodeName === 'HTML' || fromNode.nodeName === 'BODY') {
                    var toNodeHtml = toNode
                    toNode = doc.createElement('html')
                    toNode.innerHTML = toNodeHtml
                } else {
                    toNode = toElement(toNode)
                }
            } else if (toNode.nodeType === DOCUMENT_FRAGMENT_NODE$1) {
                toNode = toNode.firstElementChild
            }
            var getNodeKey = options.getNodeKey || defaultGetNodeKey
            var onBeforeNodeAdded = options.onBeforeNodeAdded || noop
            var onNodeAdded = options.onNodeAdded || noop
            var onBeforeElUpdated = options.onBeforeElUpdated || noop
            var onElUpdated = options.onElUpdated || noop
            var onBeforeNodeDiscarded = options.onBeforeNodeDiscarded || noop
            var onNodeDiscarded = options.onNodeDiscarded || noop
            var onBeforeElChildrenUpdated = options.onBeforeElChildrenUpdated || noop
            var skipFromChildren = options.skipFromChildren || noop
            var addChild =
                options.addChild ||
                function (parent, child) {
                    return parent.appendChild(child)
                }
            var childrenOnly = options.childrenOnly === true
            var fromNodesLookup = Object.create(null)
            var keyedRemovalList = []
            function addKeyedRemoval(key) {
                keyedRemovalList.push(key)
            }
            function walkDiscardedChildNodes(node, skipKeyedNodes) {
                if (node.nodeType === ELEMENT_NODE) {
                    var curChild = node.firstChild
                    while (curChild) {
                        var key = undefined
                        if (skipKeyedNodes && (key = getNodeKey(curChild))) {
                            addKeyedRemoval(key)
                        } else {
                            onNodeDiscarded(curChild)
                            if (curChild.firstChild) {
                                walkDiscardedChildNodes(curChild, skipKeyedNodes)
                            }
                        }
                        curChild = curChild.nextSibling
                    }
                }
            }
            function removeNode(node, parentNode, skipKeyedNodes) {
                if (onBeforeNodeDiscarded(node) === false) {
                    return
                }
                if (parentNode) {
                    parentNode.removeChild(node)
                }
                onNodeDiscarded(node)
                walkDiscardedChildNodes(node, skipKeyedNodes)
            }
            function indexTree(node) {
                if (node.nodeType === ELEMENT_NODE || node.nodeType === DOCUMENT_FRAGMENT_NODE$1) {
                    var curChild = node.firstChild
                    while (curChild) {
                        var key = getNodeKey(curChild)
                        if (key) {
                            fromNodesLookup[key] = curChild
                        }
                        indexTree(curChild)
                        curChild = curChild.nextSibling
                    }
                }
            }
            indexTree(fromNode)
            function handleNodeAdded(el) {
                onNodeAdded(el)
                var curChild = el.firstChild
                while (curChild) {
                    var nextSibling = curChild.nextSibling
                    var key = getNodeKey(curChild)
                    if (key) {
                        var unmatchedFromEl = fromNodesLookup[key]
                        if (unmatchedFromEl && compareNodeNames(curChild, unmatchedFromEl)) {
                            curChild.parentNode.replaceChild(unmatchedFromEl, curChild)
                            morphEl(unmatchedFromEl, curChild)
                        } else {
                            handleNodeAdded(curChild)
                        }
                    } else {
                        handleNodeAdded(curChild)
                    }
                    curChild = nextSibling
                }
            }
            function cleanupFromEl(fromEl, curFromNodeChild, curFromNodeKey) {
                while (curFromNodeChild) {
                    var fromNextSibling = curFromNodeChild.nextSibling
                    if ((curFromNodeKey = getNodeKey(curFromNodeChild))) {
                        addKeyedRemoval(curFromNodeKey)
                    } else {
                        removeNode(curFromNodeChild, fromEl, true)
                    }
                    curFromNodeChild = fromNextSibling
                }
            }
            function morphEl(fromEl, toEl, childrenOnly) {
                var toElKey = getNodeKey(toEl)
                if (toElKey) {
                    delete fromNodesLookup[toElKey]
                }
                if (!childrenOnly) {
                    var beforeUpdateResult = onBeforeElUpdated(fromEl, toEl)
                    if (beforeUpdateResult === false) {
                        return
                    } else if (beforeUpdateResult instanceof HTMLElement) {
                        fromEl = beforeUpdateResult
                        indexTree(fromEl)
                    }
                    morphAttrs(fromEl, toEl)
                    onElUpdated(fromEl)
                    if (onBeforeElChildrenUpdated(fromEl, toEl) === false) {
                        return
                    }
                }
                if (fromEl.nodeName !== 'TEXTAREA') {
                    morphChildren(fromEl, toEl)
                } else {
                    specialElHandlers.TEXTAREA(fromEl, toEl)
                }
            }
            function morphChildren(fromEl, toEl) {
                var skipFrom = skipFromChildren(fromEl, toEl)
                var curToNodeChild = toEl.firstChild
                var curFromNodeChild = fromEl.firstChild
                var curToNodeKey
                var curFromNodeKey
                var fromNextSibling
                var toNextSibling
                var matchingFromEl
                outer: while (curToNodeChild) {
                    toNextSibling = curToNodeChild.nextSibling
                    curToNodeKey = getNodeKey(curToNodeChild)
                    while (!skipFrom && curFromNodeChild) {
                        fromNextSibling = curFromNodeChild.nextSibling
                        if (curToNodeChild.isSameNode && curToNodeChild.isSameNode(curFromNodeChild)) {
                            curToNodeChild = toNextSibling
                            curFromNodeChild = fromNextSibling
                            continue outer
                        }
                        curFromNodeKey = getNodeKey(curFromNodeChild)
                        var curFromNodeType = curFromNodeChild.nodeType
                        var isCompatible = undefined
                        if (curFromNodeType === curToNodeChild.nodeType) {
                            if (curFromNodeType === ELEMENT_NODE) {
                                if (curToNodeKey) {
                                    if (curToNodeKey !== curFromNodeKey) {
                                        if ((matchingFromEl = fromNodesLookup[curToNodeKey])) {
                                            if (fromNextSibling === matchingFromEl) {
                                                isCompatible = false
                                            } else {
                                                fromEl.insertBefore(matchingFromEl, curFromNodeChild)
                                                if (curFromNodeKey) {
                                                    addKeyedRemoval(curFromNodeKey)
                                                } else {
                                                    removeNode(curFromNodeChild, fromEl, true)
                                                }
                                                curFromNodeChild = matchingFromEl
                                                curFromNodeKey = getNodeKey(curFromNodeChild)
                                            }
                                        } else {
                                            isCompatible = false
                                        }
                                    }
                                } else if (curFromNodeKey) {
                                    isCompatible = false
                                }
                                isCompatible =
                                    isCompatible !== false && compareNodeNames(curFromNodeChild, curToNodeChild)
                                if (isCompatible) {
                                    morphEl(curFromNodeChild, curToNodeChild)
                                }
                            } else if (curFromNodeType === TEXT_NODE || curFromNodeType == COMMENT_NODE) {
                                isCompatible = true
                                if (curFromNodeChild.nodeValue !== curToNodeChild.nodeValue) {
                                    curFromNodeChild.nodeValue = curToNodeChild.nodeValue
                                }
                            }
                        }
                        if (isCompatible) {
                            curToNodeChild = toNextSibling
                            curFromNodeChild = fromNextSibling
                            continue outer
                        }
                        if (curFromNodeKey) {
                            addKeyedRemoval(curFromNodeKey)
                        } else {
                            removeNode(curFromNodeChild, fromEl, true)
                        }
                        curFromNodeChild = fromNextSibling
                    }
                    if (
                        curToNodeKey &&
                        (matchingFromEl = fromNodesLookup[curToNodeKey]) &&
                        compareNodeNames(matchingFromEl, curToNodeChild)
                    ) {
                        if (!skipFrom) {
                            addChild(fromEl, matchingFromEl)
                        }
                        morphEl(matchingFromEl, curToNodeChild)
                    } else {
                        var onBeforeNodeAddedResult = onBeforeNodeAdded(curToNodeChild)
                        if (onBeforeNodeAddedResult !== false) {
                            if (onBeforeNodeAddedResult) {
                                curToNodeChild = onBeforeNodeAddedResult
                            }
                            if (curToNodeChild.actualize) {
                                curToNodeChild = curToNodeChild.actualize(fromEl.ownerDocument || doc)
                            }
                            addChild(fromEl, curToNodeChild)
                            handleNodeAdded(curToNodeChild)
                        }
                    }
                    curToNodeChild = toNextSibling
                    curFromNodeChild = fromNextSibling
                }
                cleanupFromEl(fromEl, curFromNodeChild, curFromNodeKey)
                var specialElHandler = specialElHandlers[fromEl.nodeName]
                if (specialElHandler) {
                    specialElHandler(fromEl, toEl)
                }
            }
            var morphedNode = fromNode
            var morphedNodeType = morphedNode.nodeType
            var toNodeType = toNode.nodeType
            if (!childrenOnly) {
                if (morphedNodeType === ELEMENT_NODE) {
                    if (toNodeType === ELEMENT_NODE) {
                        if (!compareNodeNames(fromNode, toNode)) {
                            onNodeDiscarded(fromNode)
                            morphedNode = moveChildren(fromNode, createElementNS(toNode.nodeName, toNode.namespaceURI))
                        }
                    } else {
                        morphedNode = toNode
                    }
                } else if (morphedNodeType === TEXT_NODE || morphedNodeType === COMMENT_NODE) {
                    if (toNodeType === morphedNodeType) {
                        if (morphedNode.nodeValue !== toNode.nodeValue) {
                            morphedNode.nodeValue = toNode.nodeValue
                        }
                        return morphedNode
                    } else {
                        morphedNode = toNode
                    }
                }
            }
            if (morphedNode === toNode) {
                onNodeDiscarded(fromNode)
            } else {
                if (toNode.isSameNode && toNode.isSameNode(morphedNode)) {
                    return
                }
                morphEl(morphedNode, toNode, childrenOnly)
                if (keyedRemovalList) {
                    for (var i = 0, len = keyedRemovalList.length; i < len; i++) {
                        var elToRemove = fromNodesLookup[keyedRemovalList[i]]
                        if (elToRemove) {
                            removeNode(elToRemove, elToRemove.parentNode, false)
                        }
                    }
                }
            }
            if (!childrenOnly && morphedNode !== fromNode && fromNode.parentNode) {
                if (morphedNode.actualize) {
                    morphedNode = morphedNode.actualize(fromNode.ownerDocument || doc)
                }
                fromNode.parentNode.replaceChild(morphedNode, fromNode)
            }
            return morphedNode
        }
    }
    var morphdom = morphdomFactory(morphAttrs)
    return morphdom
})

// Dashboard state - these are set by the template
// const TLD = '{{.TLD}}';
// const PORT = {{.Port}};
// const INITIAL_DATA = {{.InitialData}};
var currentApps = []
var expandedLogs = null
var eventSource = null
var claudeEnabled = false

// Check if Claude Code integration is configured
fetch('/api/claude-enabled')
    .then(function (r) {
        return r.json()
    })
    .then(function (data) {
        claudeEnabled = data.enabled
    })
    .catch(function () {
        claudeEnabled = false
    })

// Convert name to URL-safe slug (spaces to dashes, lowercase)
function slugify(name) {
    return name.toLowerCase().replace(/ /g, '-')
}

// Fix URL to use current protocol (http/https)
function fixProtocol(url) {
    if (!url) return url
    return url.replace(/^https?:/, window.location.protocol)
}

// Tooltip helper - returns data-tooltip attribute string
function tt(text) {
    return 'data-tooltip="' + text + '"'
}

// Icon button helper - generates button HTML with tooltip
function iconBtn(opts) {
    var classes = [opts.className]
    if (opts.visible === false) classes.push('hidden')
    return (
        '<button class="' +
        classes.join(' ') +
        '" ' +
        'onclick="' +
        opts.onclick +
        '" ' +
        tt(opts.tooltip) +
        '>' +
        opts.icon +
        '</button>'
    )
}

// Theme management
function getTheme() {
    return document.documentElement.getAttribute('data-theme') || 'system'
}

function applyTheme(theme) {
    if (theme === 'system') {
        document.documentElement.removeAttribute('data-theme')
    } else {
        document.documentElement.setAttribute('data-theme', theme)
    }
    updateThemeIcon()
}

function setTheme(theme) {
    // Save to server (which broadcasts to other pages)
    fetch('/api/theme', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ theme: theme }),
    })
    applyTheme(theme)
}

function toggleTheme() {
    var current = getTheme()
    var themes = ['system', 'light', 'dark']
    var next = themes[(themes.indexOf(current) + 1) % themes.length]
    setTheme(next)
}

function updateThemeIcon() {
    var theme = getTheme()
    var icon = document.getElementById('theme-icon')
    if (theme === 'light') icon.textContent = '☀'
    else if (theme === 'dark') icon.textContent = '☾'
    else icon.textContent = '◐'
}

// Initialize theme
setTheme(getTheme())

// SSE Connection
function connectSSE() {
    if (eventSource) {
        eventSource.close()
    }

    eventSource = new EventSource('/api/events')

    eventSource.onopen = function () {
        document.getElementById('connection-dot').classList.add('connected')
        document.getElementById('connection-text').textContent = 'Live'
    }

    eventSource.onmessage = function (event) {
        try {
            var data = JSON.parse(event.data)
            if (data.type === 'theme') {
                applyTheme(data.theme)
            } else if (Array.isArray(data)) {
                updateApps(data)
            }
        } catch (e) {
            console.error('Failed to parse SSE data:', e)
        }
    }

    eventSource.onerror = function () {
        document.getElementById('connection-dot').classList.remove('connected')
        document.getElementById('connection-text').textContent = 'Reconnecting...'
        // EventSource will auto-reconnect
    }
}

// Update apps using morphdom for efficient DOM diffing
function updateApps(newApps) {
    var container = document.getElementById('apps')
    var oldAppsSet = new Set(
        currentApps.map(function (a) {
            return a.name
        }),
    )

    // Generate new HTML
    var newHTML
    if (!newApps.length) {
        newHTML =
            '<div class="empty-state">' +
            '<h2>No apps configured</h2>' +
            '<p>Create a config file in <code style="display:inline;padding:2px 6px;margin:0">~/.config/roost-dev/</code></p>' +
            '<code># Example: create a config for your app\n' +
            'echo "npm run dev" > ~/.config/roost-dev/myapp\n\n' +
            '# Your command receives a $PORT env var\n' +
            '# Then visit http://myapp.' +
            TLD +
            '\n\n' +
            '# For more options, see: roost-dev serve --help</code>' +
            '<p style="margin-top:20px;font-size:13px">Config directory: <code style="display:inline;padding:2px 6px;margin:0">~/.config/roost-dev/</code></p>' +
            '</div>'
    } else {
        newHTML = newApps
            .map(function (app) {
                return renderApp(app)
            })
            .join('')
    }

    // Track which apps are newly added for highlight animation (skip on initial load)
    var isInitialLoad = currentApps.length === 0
    var addedApps = isInitialLoad
        ? []
        : newApps
              .filter(function (a) {
                  return !oldAppsSet.has(a.name)
              })
              .map(function (a) {
                  return a.name
              })

    // Use morphdom to efficiently update the DOM
    var wrapper = document.createElement('main')
    wrapper.id = 'apps'
    wrapper.innerHTML = newHTML

    morphdom(container, wrapper, {
        // Preserve scroll position and state
        onBeforeElUpdated: function (fromEl, toEl) {
            // Skip updating visible logs panels entirely (preserves button state, scroll, content)
            if (fromEl.classList && fromEl.classList.contains('logs-panel') && fromEl.classList.contains('visible')) {
                return false
            }
            // Preserve menu visibility
            if (fromEl.classList && fromEl.classList.contains('status-menu') && fromEl.classList.contains('visible')) {
                toEl.classList.add('visible')
            }
            return true
        },
        onNodeAdded: function (node) {
            // Highlight newly added apps
            if (node.classList && node.classList.contains('app')) {
                var name = node.getAttribute('data-name')
                if (addedApps.indexOf(name) !== -1) {
                    node.classList.add('highlight')
                    setTimeout(function () {
                        node.classList.remove('highlight')
                    }, 1000)
                }
            }
            return node
        },
    })

    currentApps = newApps

    // Re-apply filter after DOM update
    if (typeof applyFilter === 'function') {
        applyFilter()
    }
}

function renderApp(app) {
    var isRunning =
        app.running ||
        (app.services &&
            app.services.some(function (s) {
                return s.running
            }))
    var isStarting =
        app.starting ||
        (app.services &&
            app.services.some(function (s) {
                return s.starting
            }))
    var hasFailed =
        app.failed ||
        (app.services &&
            app.services.some(function (s) {
                return s.failed
            }))
    var statusClass = hasFailed ? 'failed' : isRunning ? 'running' : isStarting ? 'starting' : 'idle'
    var displayName = app.description || app.name

    var getServiceStatus = function (svc) {
        return svc.failed ? 'failed' : svc.running ? 'running' : svc.starting ? 'starting' : 'idle'
    }

    var servicesHTML = ''
    if (app.services && app.services.length > 0) {
        servicesHTML =
            '<div class="services">' +
            app.services
                .map(function (svc) {
                    var svcStatus = getServiceStatus(svc)
                    var svcTooltip =
                        { failed: 'Failed', running: 'Running', starting: 'Starting', idle: 'Idle' }[svcStatus] || ''
                    var svcSlug = slugify(svc.name)
                    var svcName = svcSlug + '-' + app.name
                    return (
                        '<div class="service">' +
                        '<div class="service-info">' +
                        '<div class="status-dot-wrapper">' +
                        '<div class="status-dot ' +
                        svcStatus +
                        '" data-tooltip="' +
                        svcTooltip +
                        '" onclick="event.stopPropagation(); handleDotClick(\'' +
                        svcName +
                        '\', event)"></div>' +
                        '<div class="status-menu" id="menu-' +
                        svcName +
                        '-active">' +
                        '<button onclick="event.stopPropagation(); doRestart(\'' +
                        svcName +
                        '\', event)">Restart</button>' +
                        '<button class="danger" onclick="event.stopPropagation(); doStop(\'' +
                        svcName +
                        '\')">Stop</button>' +
                        '</div>' +
                        '<div class="status-menu" id="menu-' +
                        svcName +
                        '-failed">' +
                        '<button onclick="event.stopPropagation(); doRestart(\'' +
                        svcName +
                        '\', event)">Restart</button>' +
                        '<button onclick="event.stopPropagation(); doClear(\'' +
                        svcName +
                        '\')">Clear</button>' +
                        '</div>' +
                        '</div>' +
                        '<span class="service-name">' +
                        svc.name +
                        '</span>' +
                        (svc.error ? '<span class="app-error">' + svc.error + '</span>' : '') +
                        '</div>' +
                        '<div class="service-meta">' +
                        '<span class="app-port">' +
                        (svc.port ? ':' + svc.port : '') +
                        '</span>' +
                        '<span class="app-uptime">' +
                        (svc.uptime || '') +
                        '</span>' +
                        '<a class="app-url" href="' +
                        fixProtocol(svc.url) +
                        '" target="_blank" rel="noopener">' +
                        svc.url.replace(/^https?:\/\//, '') +
                        '</a>' +
                        '</div>' +
                        '</div>'
                    )
                })
                .join('') +
            '</div>'
    }

    var statusTooltip = { failed: 'Failed', running: 'Running', starting: 'Starting', idle: 'Idle' }[statusClass] || ''

    var statusIndicator =
        app.type === 'static'
            ? '<div class="status-placeholder"></div>'
            : '<div class="status-dot-wrapper">' +
              '<div class="status-dot ' +
              statusClass +
              '" data-tooltip="' +
              statusTooltip +
              '" onclick="event.stopPropagation(); handleDotClick(\'' +
              app.name +
              '\', event)"></div>' +
              '<div class="status-menu" id="menu-' +
              app.name +
              '-active">' +
              '<button onclick="event.stopPropagation(); doRestart(\'' +
              app.name +
              '\', event)">Restart</button>' +
              '<button class="danger" onclick="event.stopPropagation(); doStop(\'' +
              app.name +
              '\')">Stop</button>' +
              '</div>' +
              '<div class="status-menu" id="menu-' +
              app.name +
              '-failed">' +
              '<button onclick="event.stopPropagation(); doRestart(\'' +
              app.name +
              '\', event)">Restart</button>' +
              '<button onclick="event.stopPropagation(); doClear(\'' +
              app.name +
              '\')">Clear</button>' +
              '</div>' +
              '</div>'

    return (
        '<div class="app" data-name="' +
        app.name +
        '">' +
        '<div class="app-header" onclick="toggleLogs(\'' +
        app.name +
        '\')">' +
        '<div class="app-info">' +
        statusIndicator +
        '<span class="app-name">' +
        displayName +
        '</span>' +
        (app.aliases && app.aliases.length
            ? '<span class="app-aliases">aka ' + app.aliases.join(', ') + '</span>'
            : '') +
        '</div>' +
        '<div class="app-meta">' +
        '<span class="app-port">' +
        (app.port ? ':' + app.port : '') +
        '</span>' +
        '<span class="app-uptime">' +
        (app.uptime || '') +
        '</span>' +
        (!(app.services && app.services.length) ||
        (app.services &&
            app.services.some(function (s) {
                return s.default
            }))
            ? '<a class="app-url" href="' +
              fixProtocol(app.url) +
              '" target="_blank" rel="noopener" onclick="event.stopPropagation()">' +
              app.name +
              '.' +
              TLD +
              '</a>'
            : '') +
        '</div>' +
        '</div>' +
        servicesHTML +
        '<div class="logs-panel" id="logs-' +
        app.name +
        '">' +
        '<div class="logs-header">' +
        '<span class="logs-title">Logs</span>' +
        '<div class="logs-actions" id="logs-actions-' +
        app.name +
        '">' +
        iconBtn({
            className: 'copy-btn',
            onclick: "event.stopPropagation(); copyLogs('" + app.name + "', event)",
            tooltip: 'Copy logs',
            icon: ICONS.clipboard,
        }) +
        (hasFailed
            ? iconBtn({
                  className: 'agent-btn',
                  onclick: "event.stopPropagation(); copyForAgent('" + app.name + "', event)",
                  tooltip: 'Copy for agent',
                  icon: ICONS.clipboardAgent,
              })
            : '') +
        (hasFailed && claudeEnabled
            ? iconBtn({
                  className: 'claude-btn',
                  onclick: "event.stopPropagation(); fixWithClaudeCode('" + app.name + "', event)",
                  tooltip: 'Fix with Claude Code',
                  icon: ICONS.claude,
              })
            : '') +
        iconBtn({
            className: 'clear-btn',
            onclick: "event.stopPropagation(); clearLogs('" + app.name + "')",
            tooltip: 'Clear logs',
            icon: ICONS.trash,
        }) +
        '</div>' +
        '</div>' +
        '<div class="logs-content" id="logs-content-' +
        app.name +
        '"></div>' +
        '</div>' +
        '</div>'
    )
}

function toggleLogs(name) {
    var panel = document.getElementById('logs-' + name)
    var isVisible = panel.classList.contains('visible')

    // Hide all panels
    document.querySelectorAll('.logs-panel').forEach(function (p) {
        p.classList.remove('visible')
    })

    if (!isVisible) {
        panel.classList.add('visible')
        expandedLogs = name
        // Check if this app has failed to trigger AI analysis
        var app = currentApps.find(function (a) {
            return a.name === name
        })
        var hasFailed =
            app &&
            (app.failed ||
                (app.services &&
                    app.services.some(function (s) {
                        return s.failed
                    })))
        fetchLogs(name, hasFailed)
    } else {
        expandedLogs = null
    }
}

// Check if user has selected text within an element
function hasSelectionIn(el) {
    var sel = window.getSelection()
    if (!sel || sel.isCollapsed || !sel.rangeCount) return false
    var range = sel.getRangeAt(0)
    return el.contains(range.commonAncestorContainer)
}

// Convert ANSI escape codes to HTML spans
function ansiToHtml(text) {
    var colors = {
        30: '#000',
        31: '#e74c3c',
        32: '#2ecc71',
        33: '#f1c40f',
        34: '#3498db',
        35: '#9b59b6',
        36: '#1abc9c',
        37: '#ecf0f1',
        90: '#7f8c8d',
        91: '#e74c3c',
        92: '#2ecc71',
        93: '#f1c40f',
        94: '#3498db',
        95: '#9b59b6',
        96: '#1abc9c',
        97: '#fff',
    }
    var result = ''
    var i = 0
    var openSpans = 0
    while (i < text.length) {
        if (text[i] === '\x1b' && text[i + 1] === '[') {
            var j = i + 2
            while (j < text.length && text[j] !== 'm') j++
            var codes = text.slice(i + 2, j).split(';')
            i = j + 1
            for (var k = 0; k < codes.length; k++) {
                var code = codes[k]
                if (code === '0' || code === '39' || code === '22' || code === '23') {
                    if (openSpans > 0) {
                        result += '</span>'
                        openSpans--
                    }
                } else if (colors[code]) {
                    result += '<span style="color:' + colors[code] + '">'
                    openSpans++
                } else if (code === '1') {
                    result += '<span style="font-weight:bold">'
                    openSpans++
                } else if (code === '3') {
                    result += '<span style="font-style:italic">'
                    openSpans++
                }
            }
        } else {
            // Escape HTML
            var c = text[i]
            if (c === '<') result += '&lt;'
            else if (c === '>') result += '&gt;'
            else if (c === '&') result += '&amp;'
            else result += c
            i++
        }
    }
    while (openSpans-- > 0) result += '</span>'
    return result
}

// Analyze logs with Ollama to find error lines (async, updates UI when done)
function analyzeLogsWithAI(name, lines) {
    fetch('/api/analyze-logs?name=' + encodeURIComponent(name))
        .then(function (res) {
            return res.json()
        })
        .then(function (data) {
            if (!data.enabled || data.error || !data.errorLines || data.errorLines.length === 0) {
                return
            }
            var content = document.getElementById('logs-content-' + name)
            if (!content || hasSelectionIn(content)) return
            var errorSet = new Set(data.errorLines)
            var highlighted = lines
                .map(function (line, idx) {
                    var html = ansiToHtml(line)
                    return errorSet.has(idx) ? '<mark>' + html + '</mark>' : html
                })
                .join('\n')
            content.innerHTML = highlighted
        })
        .catch(function (e) {
            console.log('AI analysis skipped:', e)
        })
}

function fetchLogs(name, triggerAnalysis) {
    fetch('/api/logs?name=' + encodeURIComponent(name))
        .then(function (res) {
            return res.json()
        })
        .then(function (lines) {
            var content = document.getElementById('logs-content-' + name)
            if (content) {
                // Skip update if user is selecting text in logs
                if (hasSelectionIn(content)) return
                var wasAtBottom = content.scrollHeight - content.scrollTop <= content.clientHeight + 50
                content.innerHTML = ansiToHtml((lines || []).join('\n'))
                if (wasAtBottom) {
                    content.scrollTop = content.scrollHeight
                }
                // Show/hide buttons based on log content
                updateLogButtons(name, lines && lines.length > 0)
                // Trigger AI analysis for failed apps
                if (triggerAnalysis && lines && lines.length > 0) {
                    analyzeLogsWithAI(name, lines)
                }
            }
        })
        .catch(function (e) {
            console.error('Failed to fetch logs:', e)
        })
}

function updateLogButtons(name, hasLogs) {
    var actions = document.getElementById('logs-actions-' + name)
    if (!actions) return
    var copyBtn = actions.querySelector('.copy-btn')
    var clearBtn = actions.querySelector('.clear-btn')
    var agentBtn = actions.querySelector('.agent-btn')
    var claudeBtn = actions.querySelector('.claude-btn')
    if (copyBtn) copyBtn.classList.toggle('visible', hasLogs)
    if (clearBtn) clearBtn.classList.toggle('visible', hasLogs)
    if (agentBtn) agentBtn.classList.toggle('visible', hasLogs)
    if (claudeBtn) claudeBtn.classList.toggle('visible', hasLogs)
}

function clearLogs(name) {
    var content = document.getElementById('logs-content-' + name)
    if (content) content.textContent = ''
    updateLogButtons(name, false)
}

function getLogsText(name) {
    var content = document.getElementById('logs-content-' + name)
    // Use selection if user has selected text within logs
    if (hasSelectionIn(content)) {
        return window.getSelection().toString()
    }
    return content.textContent
}

function copyLogs(name, event) {
    var text = getLogsText(name)

    // We use execCommand('copy') instead of navigator.clipboard.writeText() because
    // the Clipboard API requires a "secure context" (HTTPS or literal localhost).
    // Even though *.test domains resolve to 127.0.0.1, browsers check the hostname,
    // not the resolved IP - so roost-dev.test is considered insecure.
    // execCommand('copy') is deprecated but works in non-secure contexts by copying
    // selected text from a DOM element (hence the hidden textarea trick).
    var textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)

    var btn = event.target.closest('button')
    var origHTML = btn.innerHTML
    btn.innerHTML = ICONS.checkGreen
    setTimeout(function () {
        btn.innerHTML = origHTML
    }, 500)
}

function copyForAgent(name, event) {
    var logs = getLogsText(name)
    var app = currentApps.find(function (a) {
        return a.name === name
    })
    var hasFailed =
        app &&
        (app.failed ||
            (app.services &&
                app.services.some(function (s) {
                    return s.failed
                })))

    var bt = String.fromCharCode(96)
    var context =
        'I am using roost-dev, a local development server that manages apps via config files in ~/.config/roost-dev/.\n\n'
    if (hasFailed) {
        context += 'The app "' + name + '" failed to start. '
    } else {
        context += 'The app "' + name + '" is having issues. '
    }
    context +=
        'The config file is at:\n~/.config/roost-dev/' +
        name +
        '.yml\n\n' +
        'Here are the logs:\n\n' +
        bt +
        bt +
        bt +
        '\n' +
        logs +
        '\n' +
        bt +
        bt +
        bt +
        '\n\n' +
        'To restart the app: roost-dev restart ' +
        name +
        '\n' +
        'For full documentation: roost-dev docs\n\n' +
        'Please help me understand and fix this error.'

    var textarea = document.createElement('textarea')
    textarea.value = context
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)

    var btn = event.target.closest('button')
    var origHTML = btn.innerHTML
    btn.innerHTML = ICONS.checkGreen
    setTimeout(function () {
        btn.innerHTML = origHTML
    }, 500)
}

function fixWithClaudeCode(name, event) {
    var btn = event.target.closest('button')
    var origHTML = btn.innerHTML
    btn.disabled = true
    fetch('/api/open-terminal?name=' + encodeURIComponent(name))
        .then(function (res) {
            if (!res.ok) {
                return res.text().then(function (err) {
                    console.error('Failed to open terminal:', err)
                    btn.innerHTML = ICONS.xRed
                    setTimeout(function () {
                        btn.innerHTML = origHTML
                        btn.disabled = false
                    }, 2000)
                })
            }
            btn.innerHTML = ICONS.checkGreen
            setTimeout(function () {
                btn.innerHTML = origHTML
                btn.disabled = false
            }, 1000)
        })
        .catch(function (e) {
            console.error('Failed to open terminal:', e)
            btn.innerHTML = ICONS.xRed
            setTimeout(function () {
                btn.innerHTML = origHTML
                btn.disabled = false
            }, 2000)
        })
}

function stop(name) {
    return fetch('/api/stop?name=' + encodeURIComponent(name))
}

function restart(name) {
    return fetch('/api/restart?name=' + encodeURIComponent(name))
}

function start(name) {
    return fetch('/api/start?name=' + encodeURIComponent(name))
}

function closeAllMenus() {
    document.querySelectorAll('.status-menu').forEach(function (m) {
        m.classList.remove('visible')
    })
}

function handleDotClick(name, event) {
    event.stopPropagation()
    closeAllMenus()

    var dot = event.target
    var isRunning = dot.classList.contains('running')
    var isStarting = dot.classList.contains('starting')
    var isFailed = dot.classList.contains('failed')
    var menu

    if (isRunning || isStarting) {
        menu = document.getElementById('menu-' + name + '-active')
        if (menu) menu.classList.add('visible')
    } else if (isFailed) {
        menu = document.getElementById('menu-' + name + '-failed')
        if (menu) menu.classList.add('visible')
    } else {
        dot.className = 'status-dot starting'
        start(name)
    }
}

function doRestart(name, event) {
    closeAllMenus()
    var wrapper = event.target.closest('.status-dot-wrapper')
    var dot = wrapper ? wrapper.querySelector('.status-dot') : null
    if (dot) dot.className = 'status-dot starting'
    return restart(name)
}

function doStop(name) {
    closeAllMenus()
    return stop(name)
}

function doClear(name) {
    closeAllMenus()
    // Stop clears the failed state and turns it grey
    return stop(name)
}

document.addEventListener('click', closeAllMenus)

// Periodically refresh logs if panel is open
setInterval(function () {
    if (expandedLogs) {
        fetchLogs(expandedLogs)
    }
}, 2000)

// Filter functionality
var filterInput = document.getElementById('filter-input')
var filterBar = document.getElementById('filter-bar')
var currentFilter = ''

// Focus filter on '/' key
document.addEventListener('keydown', function (e) {
    // Ignore if already in an input
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
        // Handle Escape to clear and blur
        if (e.key === 'Escape') {
            filterInput.value = ''
            currentFilter = ''
            filterInput.blur()
            filterBar.classList.remove('has-value')
            applyFilter()
        }
        return
    }
    if (e.key === '/') {
        e.preventDefault()
        filterInput.focus()
    }
})

filterInput.addEventListener('input', function () {
    currentFilter = filterInput.value.toLowerCase().trim()
    filterBar.classList.toggle('has-value', currentFilter.length > 0)
    applyFilter()
})

filterInput.addEventListener('focus', function () {
    filterBar.classList.add('active')
})

filterInput.addEventListener('blur', function () {
    if (!currentFilter) {
        filterBar.classList.remove('active')
    }
})

// Normalize text for matching: lowercase, treat spaces/underscores/dashes as same
function normalizeForSearch(text) {
    return text.toLowerCase().replace(/[-_ ]/g, '')
}

// Check if normalized text contains normalized query
function matchesFilter(text, normalizedQuery) {
    if (!normalizedQuery) return true
    return normalizeForSearch(text).indexOf(normalizedQuery) !== -1
}

function applyFilter() {
    var normalizedQuery = normalizeForSearch(currentFilter)
    var apps = document.querySelectorAll('.app')

    apps.forEach(function (appEl) {
        // Gather all searchable text from visible elements
        var name = appEl.getAttribute('data-name') || ''
        var appNameEl = appEl.querySelector('.app-name')
        var aliasesEl = appEl.querySelector('.app-aliases')
        var urlEls = appEl.querySelectorAll('.app-url')
        var serviceNames = appEl.querySelectorAll('.service-name')

        var displayName = appNameEl ? appNameEl.textContent : ''
        var aliases = aliasesEl ? aliasesEl.textContent : ''

        // Check all fields for match
        var isMatch =
            !currentFilter ||
            matchesFilter(name, normalizedQuery) ||
            matchesFilter(displayName, normalizedQuery) ||
            matchesFilter(aliases, normalizedQuery)

        // Check URLs
        urlEls.forEach(function (urlEl) {
            if (matchesFilter(urlEl.textContent, normalizedQuery)) {
                isMatch = true
            }
        })

        // Check service names
        serviceNames.forEach(function (svc) {
            if (matchesFilter(svc.textContent, normalizedQuery)) {
                isMatch = true
            }
        })

        // Update classes
        appEl.classList.toggle('filter-match', isMatch && currentFilter)
        appEl.classList.toggle('filter-mismatch', !isMatch && currentFilter)

        // Highlight matching text in visible elements
        if (appNameEl) {
            appNameEl.innerHTML = highlightMatch(displayName, currentFilter)
        }
        if (aliasesEl) {
            aliasesEl.innerHTML = highlightMatch(aliasesEl.textContent, currentFilter)
        }
        serviceNames.forEach(function (svc) {
            svc.innerHTML = highlightMatch(svc.textContent, currentFilter)
        })
    })
}

function highlightMatch(text, query) {
    if (!query) return escapeHtml(text)
    // Find match using normalized comparison but highlight original positions
    var normalizedText = normalizeForSearch(text)
    var normalizedQuery = normalizeForSearch(query)
    var idx = normalizedText.indexOf(normalizedQuery)
    if (idx === -1) return escapeHtml(text)

    // Map normalized index back to original text position
    var origIdx = 0
    var normIdx = 0
    while (normIdx < idx && origIdx < text.length) {
        if (!/[-_ ]/.test(text[origIdx])) {
            normIdx++
        }
        origIdx++
    }

    // Skip leading separators at match start
    while (origIdx < text.length && /[-_ ]/.test(text[origIdx])) {
        origIdx++
    }

    // Find end position in original text
    var matchLen = 0
    var endIdx = origIdx
    while (matchLen < normalizedQuery.length && endIdx < text.length) {
        if (!/[-_ ]/.test(text[endIdx])) {
            matchLen++
        }
        endIdx++
    }

    // Trim trailing separators from match
    while (endIdx > origIdx && /[-_ ]/.test(text[endIdx - 1])) {
        endIdx--
    }

    var before = text.slice(0, origIdx)
    var match = text.slice(origIdx, endIdx)
    var after = text.slice(endIdx)
    return escapeHtml(before) + '<span class="filter-highlight">' + escapeHtml(match) + '</span>' + escapeHtml(after)
}

function escapeHtml(text) {
    return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

// Render initial data immediately, then connect SSE for updates
updateApps(INITIAL_DATA || [])
connectSSE()
