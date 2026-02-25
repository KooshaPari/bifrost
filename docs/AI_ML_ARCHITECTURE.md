# AI/ML Layer Architecture

## Overview

The Bifrost AI/ML layer provides intelligent routing, traffic analysis, behavior learning, and policy management through a multi-tier architecture combining local SLMs, cloud models, and ensemble routing.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              CLIENT REQUEST                                      │
└─────────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         FAST PATH: SEMANTIC ROUTER                               │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐                  │
│  │ ModernBERT      │  │ Embedding Cache │  │ Model Clusters  │                  │
│  │ Embeddings <3ms │  │ LRU <0.1ms      │  │ Capability Map  │                  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘                  │
│                              │                                                   │
│                    confidence > 0.85 ──────────────────────────► DIRECT ROUTE   │
│                              │                                                   │
│                    confidence < 0.85                                             │
└─────────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    BYZANTINE ENSEMBLE ROUTER (6 Voters)                          │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │                         ROUTER ENSEMBLE                                  │    │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐    │    │
│  │  │ Arch-Router  │ │ RouteLLM     │ │ MIRT-BERT    │ │ DeBERTa      │    │    │
│  │  │ 1.5B Qwen    │ │ MF Router    │ │ 25-dim IRT   │ │ Classifier   │    │    │
│  │  │ 93.17% acc   │ │ APGR 0.802   │ │ 77% OOD acc  │ │ 98.1% acc    │    │    │
│  │  │ 51ms latency │ │ 2x cost ↓    │ │ 5x cost ↓    │ │ 6-dim        │    │    │
│  │  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘    │    │
│  │  ┌──────────────┐ ┌──────────────┐                                       │    │
│  │  │ Cost-Opt     │ │ MIRT         │                                       │    │
│  │  │ Free-First   │ │ Psychometric │                                       │    │
│  │  │ Strategy     │ │ 25-latent    │                                       │    │
│  │  └──────────────┘ └──────────────┘                                       │    │
│  └─────────────────────────────────────────────────────────────────────────┘    │
│                              │                                                   │
│                    Weighted Voting (min 4/6 consensus)                           │
│                    Tolerates 2 faulty/malicious voters                           │
└─────────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           COST ENGINE                                            │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐                  │
│  │ Quota Manager   │  │ Usage Tracker   │  │ Go/No-Go        │                  │
│  │ Per-account     │  │ Real-time       │  │ Decision        │                  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘                  │
└─────────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         MODEL ENDPOINT SELECTION                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐                  │
│  │ Provider        │  │ Fallback        │  │ Health          │                  │
│  │ Accounts        │  │ Chain           │  │ Checker         │                  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘                  │
└─────────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
                              PROVIDER API CALL
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         LEARNING SYSTEM                                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐                  │
│  │ Performance     │  │ Pattern         │  │ Rule            │                  │
│  │ Tracker         │  │ Detector        │  │ Generator       │                  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘                  │
│                              │                                                   │
│                              ▼                                                   │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │                      KNOWLEDGE GRAPH (Neo4j)                             │    │
│  │  Nodes: Models, Tasks, Patterns, Rules, Behaviors                        │    │
│  │  Edges: PERFORMS_WELL_ON, SIMILAR_TO, DERIVED_FROM, VALIDATES            │    │
│  └─────────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. Local Model Roles (vibeproxy/LocalModelManager)

Each local model serves a specific role in the pipeline:

| Role | Purpose | Recommended Models | Default Port |
|------|---------|-------------------|--------------|
| **Model Router** | Routes requests to optimal cloud/local models | `katanemo/Arch-Router-1.5B`, `routellm/mf-router` | 8008 |
| **Tool Router** | Classifies and routes tool/function calls | `katanemo/Arch-Router-1.5B` | 8009 |
| **Task Classifier** | Analyzes task type and complexity | `microsoft/deberta-v3-base` | 8010 |
| **Summarizer** | Compresses context for long conversations | `mlx-community/Qwen2.5-7B-Instruct-4bit` | 8011 |
| **Code Assistant** | Primary model for code generation | `mlx-community/Qwen2.5-Coder-32B-Instruct-4bit` | 8000 |
| **Reasoner** | Complex multi-step reasoning | `mlx-community/DeepSeek-R1-Distill-Qwen-32B-4bit` | 8001 |
| **Embedder** | Generates embeddings for semantic ops | `nomic-ai/nomic-embed-text-v1.5` | 8012 |

### 2. Router Ensemble Components

#### 2.1 Arch-Router (Primary)
- **Model**: `katanemo/Arch-Router-1.5B` (Qwen 2.5-based)
- **Accuracy**: 93.17% on routing benchmarks
- **Latency**: 51ms average
- **Paper**: arXiv:2501.02141

#### 2.2 RouteLLM (Matrix Factorization)
- **Type**: Bilinear scoring with learned embeddings
- **APGR**: 0.802 (Area under Performance-Gain-Ratio curve)
- **Cost Reduction**: 2x+ while maintaining quality
- **Paper**: ICLR 2025, arXiv:2406.18665

#### 2.3 MIRT-BERT Router
- **Type**: Multidimensional Item Response Theory
- **Dimensions**: 25 latent ability dimensions
- **OOD Accuracy**: 77% (vs 70% for RouteLLM BERT)
- **Parameters**: 58K trainable (vs 110M for BERT)
- **Paper**: ACL 2025, arXiv:2506.01048

#### 2.4 NVIDIA DeBERTa Classifier
- **Accuracy**: 98.1% on complexity classification
- **Dimensions**: 6-dimensional complexity analysis
- **Task Types**: 11 categories
- **Source**: Microsoft Research

### 3. Traffic Analysis → Behavior Learning Pipeline

